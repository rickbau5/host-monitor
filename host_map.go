package hostmonitor

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

type HostMap struct {
	changes chan Change

	hosts     map[string][]*member
	hostsLock *sync.Mutex

	// configurable
	offlineTimeout time.Duration
	logger         logr.Logger
}

func NewHostMap(options ...HostMapOption) *HostMap {
	h := &HostMap{
		changes:   make(chan Change, 128),
		hosts:     make(map[string][]*member),
		hostsLock: &sync.Mutex{},

		offlineTimeout: 5 * time.Minute,
		logger:         stdr.New(log.Default()),
	}

	for _, option := range options {
		option.apply(h)
	}

	return h
}

func (h *HostMap) update(addr Addr, emitChanges bool) bool {
	mac := addr.MAC.String()
	if mac == "" {
		return false
	}

	now := time.Now()

	h.hostsLock.Lock()
	existing, ok := h.hosts[mac]
	defer h.hostsLock.Unlock()
	if !ok {
		// new host! (new to us)
		h.hosts[mac] = []*member{
			{
				addr:     addr,
				active:   true,
				lastSeen: now,
			},
		}

		if emitChanges {
			h.sendChange(Change{
				ChangeType:   OnlineChange,
				Addr:         addr,
				Online:       true,
				PreviousAddr: nil,
				LastSeen:     now,
			})
		}

		return true
	}

	// update the last seen time if we've seen this ip already
	var found bool
	var previousAddr *Addr
	for _, m := range existing {
		if m.addr.IP == addr.IP {
			// we've seen this ip before for this mac mark it as active
			m.lastSeen = now
			m.active = true
			found = true
		} else {
			if m.active {
				// this is the previous active addr for the mac
				// shallow copy
				previousAddr = new(Addr)
				*previousAddr = m.addr
			}
			// not the current ip for the mac (anymore)
			m.active = false
		}
	}

	if found && previousAddr == nil {
		// did not change addresses
		return false
	}

	if !found {
		// if we haven't seen this ip for this host add it to the member list for that mac
		h.hosts[mac] = append(existing, &member{
			addr:     addr,
			active:   true,
			lastSeen: now,
		})
	}

	if emitChanges {
		// emit a change regardless if we've seen the ip already for this mac - the device switched back
		h.sendChange(Change{
			ChangeType:   IPChange,
			Addr:         addr,
			Online:       true,
			PreviousAddr: previousAddr,
			LastSeen:     now,
		})
	}
	return true
}

func (h *HostMap) reap() bool {
	h.hostsLock.Lock()
	defer h.hostsLock.Unlock()

	var changed bool
	now := time.Now()

	for key, members := range h.hosts {
		var newMembers []*member
		for _, m := range members {
			if now.Sub(m.lastSeen) < 5*time.Minute {
				newMembers = append(newMembers, m)
				continue
			}
			changed = true

			h.sendChange(Change{
				ChangeType:   OfflineChange,
				Addr:         m.addr,
				Online:       false,
				PreviousAddr: nil,
				LastSeen:     m.lastSeen,
			})
		}

		if len(newMembers) == 0 {
			// delete the entire entry for this mac
			delete(h.hosts, key)
			continue
		}

		h.hosts[key] = newMembers
	}

	return changed
}

func (h *HostMap) sendChange(change Change) {
	select {
	case h.changes <- change:
	default:
		h.logger.Info("dropping change, notification channel full", "change", change)
	}
}

// Reset clears all the currently tracked hosts.
func (h *HostMap) Reset() {
	h.hostsLock.Lock()
	h.hosts = make(map[string][]*member)
	h.hostsLock.Unlock()
}

// ResetAndLoad resets the host map and loads the map with a list of addresses. This does not emit any changes to
// notifications. Use UpdateAddresses if notifications are required.
func (h *HostMap) ResetAndLoad(addrs []Addr) {
	h.Reset()
	for _, addr := range addrs {
		h.update(addr, false)
	}
}

// UpdateAddresses updates the existing host map with the specified addresses, emitting notifications as needed.
func (h *HostMap) UpdateAddresses(addrs []Addr) bool {
	// update with the new addresses
	var changed bool
	for _, addr := range addrs {
		changed = h.update(addr, true) || changed
	}

	// reap old entries if expired
	changed = changed || h.reap()

	return changed
}

func (h *HostMap) PrintTable() {
	h.hostsLock.Lock()
	defer h.hostsLock.Unlock()
	for _, m := range h.hosts {
		name := FindManufacturer(m[0].addr.MAC)
		if name == "" {
			name = "unknown"
		}
		h.logger.Info("current table", "mac", m[0].addr.MAC, "members", m, "manufacturer", name)
	}
}

func (h *HostMap) Notifications() <-chan Change {
	return h.changes
}

// From https://github.com/irai/packet/blob/3d13deba3c30b27bbb6da8ec122a96e45fe92a27/addr.go#L12-L16
type Addr struct {
	MAC  net.HardwareAddr
	IP   netip.Addr
	Port uint16
}

func (addr Addr) String() string {
	return fmt.Sprintf("mac=(%s) ip=(%s) port=(%d)", addr.MAC, addr.IP, addr.Port)
}

type ChangeType int

const (
	UnknownChange ChangeType = iota
	IPChange
	OnlineChange
	OfflineChange
)

func (ct ChangeType) String() string {
	switch ct {
	case IPChange:
		return "ip change"
	case OnlineChange:
		return "online"
	case OfflineChange:
		return "offline"
	default:
		return "unknown"
	}
}

type Change struct {
	ChangeType ChangeType
	Addr       Addr
	Online     bool

	PreviousAddr *Addr
	LastSeen     time.Time
}

func (c Change) String() string {
	return fmt.Sprintf("change=(%s) online=(%v) addr=(%s) previousAddr=(%s) lastSeen=(%s)",
		c.ChangeType, c.Online, c.Addr, c.PreviousAddr, c.LastSeen)
}

type member struct {
	addr     Addr
	active   bool
	lastSeen time.Time
}

func (m member) String() string {
	return fmt.Sprintf("%s lastSeen=(%s)", m.addr.String(), m.lastSeen.Format(time.RFC3339))
}

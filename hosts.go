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
	lastSeen time.Time
}

func (m member) String() string {
	return fmt.Sprintf("%s lastSeen=(%s)", m.addr.String(), m.lastSeen.Format(time.RFC3339))
}

type HostMap struct {
	changes chan Change

	logger logr.Logger

	hosts     map[string][]*member
	hostsLock *sync.Mutex
}

func NewHostMap() *HostMap {
	h := &HostMap{
		changes:   make(chan Change, 128),
		hosts:     make(map[string][]*member),
		hostsLock: &sync.Mutex{},

		logger: stdr.New(log.Default()),
	}

	return h
}

func (h *HostMap) SetLogger(logger logr.Logger) {
	h.logger = logger
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

	// update the last seen time
	var found bool
	for _, m := range existing {
		if m.addr.IP == addr.IP {
			found = true
			m.lastSeen = now
			break
		}
	}

	if found {
		return false
	}

	members := append(existing, &member{
		addr:     addr,
		lastSeen: now,
	})
	h.hosts[mac] = members

	if emitChanges {
		h.sendChange(Change{
			ChangeType:   IPChange,
			Addr:         addr,
			Online:       true,
			PreviousAddr: &existing[len(existing)-1].addr,
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

func (h *HostMap) UpdateAddresses(addrs []Addr) bool {
	// update with the new addresses
	var changed bool
	for _, addr := range addrs {
		changed = changed || h.update(addr, true)
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

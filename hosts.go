package main

import (
	"fmt"
	"github.com/irai/packet"
	"log"
	"sync"
	"time"
)

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
	Addr       packet.Addr
	Online     bool

	PreviousAddr *packet.Addr
	LastSeen     time.Time
}

func (c Change) String() string {
	return fmt.Sprintf("change=(%s) online=(%v) addr=(%s) previousAddr=(%s) lastSeen=(%s)",
		c.ChangeType, c.Online, c.PreviousAddr, c.LastSeen, c.Addr)
}

type member struct {
	addr     packet.Addr
	lastSeen time.Time
}

type HostMap struct {
	changes chan Change

	hosts     map[string][]*member
	hostsLock *sync.Mutex
}

func NewHostMap() *HostMap {
	h := &HostMap{
		changes:   make(chan Change, 128),
		hosts:     make(map[string][]*member),
		hostsLock: &sync.Mutex{},
	}

	return h
}

func (h *HostMap) Load(nic string) error {
	addrs, err := packet.LoadLinuxARPTable(nic)
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		h.update(addr, false)
	}
	return nil
}

func (h *HostMap) update(addr packet.Addr, emitChanges bool) bool {
	mac := addr.MAC.String()
	if mac == "" {
		log.Println("skipping addr with no mac:", addr.String())
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
			if now.Sub(m.lastSeen) < time.Second*15 {
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
		log.Println("dropping change, notification channel full:", change)
	}
}

func (h *HostMap) UpdateAddresses(addrs []packet.Addr) bool {
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
		name := packet.FindManufacturer(m[0].addr.MAC)
		if name == "" {
			name = "unknown"
		}
		log.Printf("%s manufacturer=%s", m[0].addr, name)
	}
}

func (h *HostMap) Notifications() <-chan Change {
	return h.changes
}

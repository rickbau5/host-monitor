package main

import (
	"net"
	"sync"
)

func NewMacHostMap() *MacHostMap {
	return &MacHostMap{
		m:   make(map[string]string),
		mux: &sync.RWMutex{},
	}
}

type MacHostMap struct {
	m   map[string]string
	mux *sync.RWMutex
}

func (m *MacHostMap) Set(mac net.HardwareAddr, hostName string) {
	m.mux.Lock()
	m.m[mac.String()] = hostName
	defer m.mux.Unlock()
}

func (m *MacHostMap) Get(mac net.HardwareAddr) string {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.m[mac.String()]
}

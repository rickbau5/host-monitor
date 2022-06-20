package main

import "github.com/google/gopacket"

func NewHandle(iface string) (gopacket.PacketDataSource, func(), error) {
	return newHandle(iface)
}

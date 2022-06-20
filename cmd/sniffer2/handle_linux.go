//go:build arm
// +build arm

package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
)

func newHandle(iface string) (gopacket.PacketDataSource, func(), error) {
	handle, err := pcapgo.NewEthernetHandle(iface)
	if err != nil {
		return nil, nil, err
	}

	if err = handle.SetCaptureLength(snapLen); err != nil {
		return nil, nil, err
	} else if err = handle.SetPromiscuous(true); err != nil {
		return nil, nil, err
	}

	// setup BPF filter

	return handle, handle.Close, nil
}

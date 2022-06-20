//go:build darwin
// +build darwin

package main

import (
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func newHandle(iface string) (gopacket.PacketDataSource, func(), error) {
	inactive, err := pcap.NewInactiveHandle(iface)
	if err != nil {
		log.Fatalf("could not create: %v", err)
	}
	defer inactive.CleanUp()
	if err = inactive.SetSnapLen(snapLen); err != nil {
		log.Fatalf("could not set snap length: %v", err)
	} else if err = inactive.SetPromisc(true); err != nil {
		log.Fatalf("could not set promisc mode: %v", err)
	} else if err = inactive.SetTimeout(time.Second); err != nil {
		log.Fatalf("could not set timeout: %v", err)
	}

	handle, err := inactive.Activate()
	if err != nil {
		log.Fatal("PCAP Activate error:", err)
	}

	// set our filter for port 68 - dhcp packets
	if err = handle.SetBPFFilter("udp port 68"); err != nil {
		log.Fatal("BPF filter error:", err)
	}

	return handle, handle.Close, nil
}

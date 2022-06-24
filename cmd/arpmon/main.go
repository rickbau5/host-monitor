package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	hostmonitor "github.com/rickbau5/host-monitor"

	"github.com/irai/packet"
)

var (
	iface string
)

func init() {
	flag.StringVar(&iface, "i", "", "the name of the network interface to load the arp table for")
}

func main() {
	flag.Parse()
	if iface == "" {
		fmt.Println("argument -i is required")
		os.Exit(1)
	}

	hosts := hostmonitor.NewHostMap()

	addrs, err := packet.LoadLinuxARPTable(iface)
	if err != nil {
		fmt.Println("failed initial load:", err)
		os.Exit(1)
	}
	hosts.UpdateAddresses(addrsToAddrs(addrs))

	hosts.PrintTable()

	go func() {
		for change := range hosts.Notifications() {
			log.Println("change detected:", change)
		}
	}()

	ticks := time.Tick(15 * time.Second)
	for range ticks {
		addrs, err := packet.LoadLinuxARPTable(iface)
		if err != nil {
			log.Println("error loading linux arp table:", err)
			continue
		}

		_ = hosts.UpdateAddresses(addrsToAddrs(addrs))
	}
}

func addrsToAddrs(addrs []packet.Addr) []hostmonitor.Addr {
	hmAddrs := make([]hostmonitor.Addr, len(addrs))
	for i, addr := range addrs {
		hmAddrs[i] = hostmonitor.Addr{
			MAC:  addr.MAC,
			IP:   addr.IP,
			Port: addr.Port,
		}
	}

	return hmAddrs
}

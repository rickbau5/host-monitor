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
	flag.StringVar(&iface, "iface", "", "the name of the network interface to load the arp table for")
}

func main() {
	flag.Parse()
	if iface == "" {
		fmt.Println("argument --iface is required")
		os.Exit(1)
	}

	hosts := hostmonitor.NewHostMap()
	if err := hosts.Load(iface); err != nil {
		fmt.Println("failed initial load:", err)
		os.Exit(1)
	}

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
			return
		}

		_ = hosts.UpdateAddresses(addrs)
	}
}

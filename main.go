package main

import (
	"flag"
	"fmt"
	"github.com/irai/packet"
	"log"
	"os"
	"time"
)

var (
	nic string
)

func init() {
	flag.StringVar(&nic, "nic", "", "the name of the network interface to load the arp table for")
}

func main() {
	flag.Parse()
	if nic == "" {
		fmt.Println("argument --nic is required")
		os.Exit(1)
	}

	hosts := NewHostMap()
	if err := hosts.Load(nic); err != nil {
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
		addrs, err := packet.LoadLinuxARPTable(nic)
		if err != nil {
			log.Println("error loading linux arp table:", err)
			return
		}

		_ = hosts.UpdateAddresses(addrs)
	}
}

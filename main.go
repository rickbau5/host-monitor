package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/irai/packet"
)

var (
	nic string
)

func init() {
	flag.StringVar(&nic, "nic", "", "set the nic to listen on")
}

func main() {
	flag.Parse()
	if nic == "" {
		fmt.Println("argument --nic is required")
		os.Exit(1)
	}

	run(nic)

	ticks := time.Tick(15 * time.Second)
	for range ticks {
		run(nic)
	}
}

func run(nic string) {
	addrs, err := packet.LoadLinuxARPTable(nic)
	if err != nil {
		fmt.Println("error loading linux arp table:", err)
		return
	}

	sort.Slice(addrs, func(i, j int) bool {
		if addrs[i].IP.Is6() && addrs[j].IP.Is6() {
			return addrs[i].IP.String() < addrs[j].IP.String()
		}
		if addrs[i].IP.Is4() && addrs[j].IP.Is4() {
			return int(addrs[i].IP.As4()[3]) < int(addrs[j].IP.As4()[3])
		}
		return addrs[i].IP.Is4()
	})

	log.Println("--- arp table ---")
	for _, addr := range addrs {
		name := packet.FindManufacturer(addr.MAC)
		if name == "" {
			name = "unknown"
		}
		log.Printf("%s manufacturer=%s", addr, name)
	}
	log.Println("--- end ---")
}

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// adapted from pcapdump example https://github.com/google/gopacket/blob/master/examples/pcapdump/main.go

var iface = flag.String("i", "", "Name of the interface to read packets from")

const (
	snapLen = 65536
)

func main() {
	flag.Parse()
	if *iface == "" {
		log.Fatal("--i required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	handle, closeFunc, err := NewHandle(*iface)
	if err != nil {
		log.Fatal("failed creating handle:", err)
	}
	defer closeFunc()

	source := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)

	log.Println("ready to read packets")
	err = readPackets(ctx, source.Packets(), handler)
	if err != nil {
		log.Fatal("error reading packets:", err)
	}

	log.Println("exiting...")
}

type Handler func(ctx context.Context, dhcp *layers.DHCPv4) error

func handler(_ context.Context, dhcp *layers.DHCPv4) error {
	manufacturer := FindManufacturer(dhcp.ClientHWAddr)
	hostName := "<unknown>"
	for _, opt := range dhcp.Options {
		if opt.Type != layers.DHCPOptHostname {
			continue
		}

		hostName = string(opt.Data)
	}

	// see http://www.tcpipguide.com/free/t_DHCPMessageFormat.htm
	// not always set
	ip := dhcp.YourClientIP
	if ip.IsUnspecified() {
		ip = dhcp.YourClientIP // won't be set unless we requested it (unless the iface supports promiscuous mode)
	}

	log.Printf("dhcp(%d) from %s(ip=%s), hostname=(%s), manufacturer=(%s)",
		dhcp.Operation, dhcp.ClientHWAddr, ip, hostName, manufacturer)

	return nil
}

func readPackets(ctx context.Context, packets <-chan gopacket.Packet, handler Handler) error {
	for {
		var (
			packet gopacket.Packet
			ok     bool
		)
		select {
		case packet, ok = <-packets:
			if !ok {
				return errors.New("packet chan closed")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		layer := packet.Layer(layers.LayerTypeDHCPv4)

		dhcp, ok := layer.(*layers.DHCPv4)
		if !ok {
			continue
		}

		if err := handler(ctx, dhcp); err != nil {
			log.Println("error handling packet:", err)
		}
	}
}

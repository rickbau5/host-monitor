package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	hostmonitor "github.com/rickbau5/host-monitor"
)

// adapted from pcapdump example https://github.com/google/gopacket/blob/master/examples/pcapdump/main.go

var iface = flag.String("i", "", "Name of the interface to read packets from")

const (
	snapLen = 65536
)

var hostNames = NewMacHostMap()

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

	hosts := hostmonitor.NewHostMap()
	go func() {
		for notification := range hosts.Notifications() {
			hostName := hostNames.Get(notification.Addr.MAC)
			log.Printf("host '%s' changed: %s", hostName, notification)
		}
	}()

	log.Println("ready to read packets")
	err = readPackets(ctx, source.Packets(), hosts, handler)
	if err != nil {
		log.Fatal("error reading packets:", err)
	}

	log.Println("exiting...")
}

type Handler func(ctx context.Context, dhcp *layers.DHCPv4) error

func handler(_ context.Context, dhcp *layers.DHCPv4) error {
	manufacturer := hostmonitor.FindManufacturer(dhcp.ClientHWAddr)
	var hostName string
	for _, opt := range dhcp.Options {
		if opt.Type != layers.DHCPOptHostname {
			continue
		}

		hostName = string(opt.Data)
	}

	if hostName != "" {
		hostNames.Set(dhcp.ClientHWAddr, hostName)
	} else {
		hostName = "<unknown>"
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

func readPackets(ctx context.Context, packets <-chan gopacket.Packet, hosts *hostmonitor.HostMap, handler Handler) error {
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

		var sourceMac, dstMac net.HardwareAddr
		sourceIp := net.IPv4zero
		if layer := packet.Layer(layers.LayerTypeEthernet); layer != nil {
			eth, _ := layer.(*layers.Ethernet)
			sourceMac = eth.SrcMAC
			dstMac = eth.DstMAC
		}
		if layer := packet.Layer(layers.LayerTypeIPv4); layer != nil {
			ipv4, _ := layer.(*layers.IPv4)
			if len(ipv4.SrcIP) > 0 && (ipv4.SrcIP.IsPrivate() || ipv4.SrcIP.IsUnspecified()) {
				sourceIp = ipv4.SrcIP
			} else if len(ipv4.SrcIP) > 0 && (ipv4.DstIP.IsPrivate() || ipv4.DstIP.IsUnspecified()) {
				sourceIp = ipv4.DstIP
				sourceMac, dstMac = dstMac, sourceMac
			}
		}

		if !sourceIp.IsPrivate() && !sourceIp.IsUnspecified() {
			// drop
			continue
		}

		if !sourceIp.IsUnspecified() {
			ip, _ := netip.AddrFromSlice(sourceIp)
			hosts.UpdateAddresses([]hostmonitor.Addr{
				{
					MAC:  sourceMac,
					IP:   ip,
					Port: 0,
				},
			})
		}

		// extract dhcp4 for host
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

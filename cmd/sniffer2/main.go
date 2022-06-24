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
	"time"

	"github.com/go-logr/stdr"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	hostmonitor "github.com/rickbau5/host-monitor"
)

// adapted from pcapdump example https://github.com/google/gopacket/blob/master/examples/pcapdump/main.go

const (
	snapLen            = 65536
	defaultOfflineTime = 5 * time.Minute
)

var iface = flag.String("i", "", "Name of the interface to read packets from")
var offlineTime = flag.Duration("offline-timeout", defaultOfflineTime, "Amount of time that must elapse before a host is considered inactive")

func main() {
	flag.Parse()
	if *iface == "" {
		log.Fatal("--i required")
	}

	if *offlineTime < 0 {
		*offlineTime = defaultOfflineTime
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	handle, closeFunc, err := NewHandle(*iface)
	if err != nil {
		log.Fatal("failed creating handle:", err)
	}
	defer closeFunc()

	source := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)

	// keep track of MAC -> IP addresses
	hosts := hostmonitor.NewHostMap(
		hostmonitor.LoggerOption(stdr.New(log.New(os.Stdout, "", log.LstdFlags))),
		hostmonitor.HostOfflineTimeoutOption(*offlineTime),
	)
	// keep track of MAC -> Host Names from DHCP
	hostNames := NewMacHostMap()

	go func() {
		// notifications are emitted when a host changes
		//  * new IP
		//  * host comes online
		//  * host goes offline
		for notification := range hosts.Notifications() {
			hostName := hostNames.Get(notification.Addr.MAC)
			log.Printf("host '%s' changed: %s", hostName, notification)
		}
	}()

	// composes the two separate handlers for handling host updates and hostname updates into a single handler
	var (
		updateHosts     = UpdateHosts(hosts)
		updateHostNames = UpdateHostNames(hostNames)
	)
	packetHandler := PacketHandler(func(ctx context.Context, packet gopacket.Packet) error {
		if err := updateHosts(ctx, packet); err != nil {
			return err
		}

		return updateHostNames(ctx, packet)
	})

	log.Println("ready to read packets")
	err = readPackets(ctx, source.Packets(), packetHandler)
	if err != nil {
		log.Fatal("error reading packets:", err)
	}

	log.Println("exiting...")
}

func readPackets(ctx context.Context, packets <-chan gopacket.Packet, handler PacketHandler) error {
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

		if err := handler(ctx, packet); err != nil {
			log.Println("error handling packet:", err)
		}
	}
}

type PacketHandler func(ctx context.Context, packet gopacket.Packet) error

// UpdateHosts keeps the provided hosts up to date with the addresses of hosts on private network.
// It's assumed we're running on a private a network like 192.168.0.0 or 10.0.0.0
func UpdateHosts(hosts *hostmonitor.HostMap) PacketHandler {
	return func(_ context.Context, packet gopacket.Packet) error {

		var sourceMac, dstMac net.HardwareAddr
		ipInNetwork := net.IPv4zero

		if layer := packet.Layer(layers.LayerTypeEthernet); layer != nil {
			eth, _ := layer.(*layers.Ethernet)
			sourceMac = eth.SrcMAC
			dstMac = eth.DstMAC
		}

		if layer := packet.Layer(layers.LayerTypeIPv4); layer != nil {
			ipv4, _ := layer.(*layers.IPv4)

			if len(ipv4.SrcIP) > 0 {
				if ipv4.SrcIP.IsPrivate() {
					// packet going out of the network from a private host
					ipInNetwork = ipv4.SrcIP
				} else if ipv4.DstIP.IsPrivate() {
					// packet coming in from outside network
					ipInNetwork = ipv4.DstIP
					sourceMac, dstMac = dstMac, sourceMac
				}
			}
		}

		// unknown ip address (could be a dhcp broadcast)
		if ipInNetwork.IsUnspecified() {
			// skip
			return nil
		}

		// safety check for private host
		if !ipInNetwork.IsPrivate() {
			// skip
			return nil
		}

		ip, _ := netip.AddrFromSlice(ipInNetwork)
		hosts.UpdateAddresses([]hostmonitor.Addr{
			{
				MAC: sourceMac,
				IP:  ip,
			},
		})

		return nil
	}
}

// UpdateHostNames watches for dhcpv4 packets and updates the MacHostMap with this names, if set.
// This also logs the dhcp packet info
func UpdateHostNames(hostNames *MacHostMap) PacketHandler {
	return func(_ context.Context, packet gopacket.Packet) error {
		// extract dhcp4 for host
		layer := packet.Layer(layers.LayerTypeDHCPv4)
		dhcp, ok := layer.(*layers.DHCPv4)
		if !ok {
			// nothing to do - not a DHCPv4 Packet
			return nil
		}

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
		log.Printf("dhcp(%d) from %s(ip=%s), hostname=(%s), manufacturer=(%s)",
			dhcp.Operation, dhcp.ClientHWAddr, dhcp.YourClientIP, hostName, manufacturer)

		return nil
	}
}

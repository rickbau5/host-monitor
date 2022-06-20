package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/irai/packet"
)

var (
	iface string
)

func init() {
	flag.StringVar(&iface, "iface", "", "set the interface to listen on")
}

func main() {
	flag.Parse()
	if iface == "" {
		fmt.Println("argument --iface is required")
		os.Exit(1)
	}

	session, err := packet.NewSession(iface)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		log.Println("session starting")
		for notification := range session.C {
			status := "offline"
			if notification.Online {
				status = "online"
			}

			log.Printf("%s (%s) is %s: %s", notification.Addr.MAC, notification.Addr.IP, status, notification)
		}
	}()

	go func() {
		buf := make([]byte, packet.EthMaxSize)
		for {
			select {
			case <-ctx.Done():
				log.Println("stopping packet processing")
				return
			default:
			}

			n, _, err := session.ReadFrom(buf)
			if err != nil {
				log.Println("failed reading from session:", err)
				continue
			}

			frame, err := session.Parse(buf[:n])
			if err != nil {
				log.Println("failed parsing buffer:", err)
				continue
			}
			switch frame.PayloadID {
			case packet.PayloadDHCP4:
				dhcpPacket, err := dhcpv4.FromBytes(frame.Payload())
				if err != nil {
					log.Println("error parsing packet:", err)
					continue
				}

				hostName := string(dhcpPacket.Options.Get(dhcpv4.OptionHostName))
				manufacturer := packet.FindManufacturer(frame.SrcAddr.MAC)
				log.Printf("dhcp(%d) from %s(ip=%s), hostname=(%s), manufacturer=(%s)",
					dhcpPacket.OpCode, frame.SrcAddr.MAC, frame.SrcAddr.IP, hostName, manufacturer)
			}
		}
	}()

	<-ctx.Done()
	session.Close()
	log.Println("stopped")
}

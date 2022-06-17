package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	packet.

	session, err := packet.NewSession(nic)
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
			session.PrintTable()
		}
	}()

	go func() {
		buf := make([]byte, 0)
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
			if n == 0 {
				time.Sleep(time.Millisecond * 250)
				continue
			}

			frame, err := session.Parse(buf[:n])
			if err != nil {
				log.Println("failed parsing buffer:", err)
				continue
			}

			log.Printf("%s -> %s: size %d", frame.SrcAddr, frame.DstAddr, n)
		}
	}()

	<-ctx.Done()
	session.Close()
	log.Println("stopped")
}

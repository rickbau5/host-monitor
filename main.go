package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/irai/packet"
	"log"
	"os"
	"os/signal"
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

	session, err := packet.NewSession(nic)
	if err != nil {
		panic(err)
	}

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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()
	session.Close()
	log.Println("stopped")
}

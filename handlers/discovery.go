package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	routingdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

func DiscoveryHandler(h host.Host, peerChan chan peer.AddrInfo, rendezvousString string) {
	// Set up mDNS
	mdnsService := mdns.NewMdnsService(h, rendezvousString, &DiscoveryNotifee{peerChan: peerChan})
	if err := mdnsService.Start(); err != nil {
		fmt.Println("Error starting mDNS service:", err)
		os.Exit(1)
	}

	// Set up DHT
	kadDHT, err := dht.New(context.Background(), h)
	if err != nil {
		fmt.Println("Error creating DHT:", err)
		os.Exit(1)
	}

	if err = kadDHT.Bootstrap(context.Background()); err != nil {
		fmt.Println("Error bootstrapping DHT:", err)
		os.Exit(1)
	}

	routingDiscovery := routingdisc.NewRoutingDiscovery(kadDHT)
	go func() {
		for {
			// Advertise the rendezvous string
			_, err := routingDiscovery.Advertise(context.Background(), rendezvousString)
			if err != nil {
				fmt.Println("Error advertising:", err)
			}

			// Find peers using the rendezvous string
			peers, err := routingDiscovery.FindPeers(context.Background(), rendezvousString)
			if err != nil {
				fmt.Println("Error finding peers:", err)
				continue
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				fmt.Println("Found peer via DHT:", p)
				peerChan <- p
			}

			time.Sleep(1 * time.Minute)
		}
	}()
}

type DiscoveryNotifee struct {
	peerChan chan peer.AddrInfo
}

func (n *DiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.peerChan <- pi
}

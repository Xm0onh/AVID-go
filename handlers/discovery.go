package handlers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	routingdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

const rendezvousString = "libp2p-mdns"

func DiscoveryHandler(h host.Host, peerChan chan peer.AddrInfo) {
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

	// Read bootstrap addresses from file
	file, err := os.Open("bootstrap_addrs.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addr := scanner.Text()
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			fmt.Println("Error parsing multiaddr:", err)
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			fmt.Println("Error converting multiaddr to peer.AddrInfo:", err)
			continue
		}
		peerChan <- *pi
		h.Connect(context.Background(), *pi)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

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

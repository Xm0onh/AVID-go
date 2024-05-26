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
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
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

	// Read bootstrap addresses from file
	file, err := os.Open("bootstrap_addrs.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	var bootstrapPeers []peer.AddrInfo
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
		bootstrapPeers = append(bootstrapPeers, *pi)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	// Set up DHT with bootstrap peers
	ctx := context.Background()
	kadDHT, err := dht.New(ctx, h, dht.BootstrapPeers(bootstrapPeers...))
	if err != nil {
		fmt.Println("Error creating DHT:", err)
		os.Exit(1)
	}

	if err = kadDHT.Bootstrap(ctx); err != nil {
		fmt.Println("Error bootstrapping DHT:", err)
		os.Exit(1)
	}

	// Log bootstrap connections
	for _, peerInfo := range bootstrapPeers {
		if err := h.Connect(ctx, peerInfo); err != nil {
			fmt.Printf("Error connecting to bootstrap peer %s: %v\n", peerInfo.ID, err)
		} else {
			fmt.Printf("Connected to bootstrap peer: %s\n", peerInfo.ID)
		}
	}

	// Wait a bit to let bootstrapping finish
	time.Sleep(1 * time.Second)

	routingDiscovery := routingdisc.NewRoutingDiscovery(kadDHT)
	dutil.Advertise(ctx, routingDiscovery, rendezvousString)
	fmt.Println("Successfully advertised!")

	go func() {
		for {
			// Find peers using the rendezvous string
			peers, err := routingDiscovery.FindPeers(ctx, rendezvousString)
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

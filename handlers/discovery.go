package handlers

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

func DiscoveryHandler(h host.Host, peerChan chan peer.AddrInfo, rendezvousString string) {
	mdnsService := mdns.NewMdnsService(h, rendezvousString, &DiscoveryNotifee{peerChan: peerChan})
	if err := mdnsService.Start(); err != nil {
		fmt.Println("Error starting mDNS service:", err)
		os.Exit(1)
	}
}

type DiscoveryNotifee struct {
	peerChan chan peer.AddrInfo
}

func (n *DiscoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.peerChan <- pi
}

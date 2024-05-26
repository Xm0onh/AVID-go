package bootstrap

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
)

func StartBootstrapNode(port int, ip string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		fmt.Println("Error generating key pair:", err)
		os.Exit(1)
	}

	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
	if err != nil {
		fmt.Println("Error creating multiaddr:", err)
		os.Exit(1)
	}

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrs(addr),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New))
	if err != nil {
		fmt.Println("Error creating libp2p host:", err)
		os.Exit(1)
	}

	fmt.Printf("Bootstrap node %s is listening on %s:%d with Peer ID %s...\n", ip, port, h.ID())

	kadDHT, err := dht.New(ctx, h)
	if err != nil {
		fmt.Println("Error creating DHT:", err)
		os.Exit(1)
	}

	if err = kadDHT.Bootstrap(ctx); err != nil {
		fmt.Println("Error bootstrapping DHT:", err)
		os.Exit(1)
	}

	peerAddr := fmt.Sprintf("%s/p2p/%s", addr, h.ID().String())

	// Append the address to a file
	file, err := os.OpenFile("bootstrap_addrs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	if _, err := file.WriteString(peerAddr + "\n"); err != nil {
		fmt.Println("Error writing to file:", err)
		os.Exit(1)
	}

	// Print the full multiaddresses for the bootstrap node
	fmt.Printf("Bootstrap node is running. ID: %s\n", h.ID().String())
	for _, addr := range h.Addrs() {
		fmt.Printf("Address: %s/p2p/%s\n", addr, h.ID().String())
	}

	// Ensure the DHT has some time to populate
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if len(kadDHT.RoutingTable().ListPeers()) > 0 {
					break
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()

	// Handle OS signals to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")
	h.Close()
}

package bootstrap

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
)

func StartBootstrapNode(port int, ip string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
	h, err := libp2p.New(libp2p.Identity(priv), libp2p.ListenAddrs(addr))
	if err != nil {
		fmt.Println("Error creating libp2p host:", err)
		os.Exit(1)
	}

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

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down...")
	h.Close()
}

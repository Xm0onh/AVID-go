package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const rendezvousString = "libp2p-mdns"
const chunkSize = 3

var receivedChunks = sync.Map{}
var sentChunks = sync.Map{}
var nodeMutex = sync.Mutex{}
var expectedChunks = 3
var connectedPeers []peer.AddrInfo
var node1ID peer.ID

func main() {
	nodeID := flag.String("node", "", "Node ID")
	flag.Parse()

	if *nodeID == "" {
		fmt.Println("Please specify node ID with -node flag.")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	h, _ := libp2p.New(libp2p.Identity(priv))

	var wg sync.WaitGroup
	peerChan := make(chan peer.AddrInfo)
	peerDataChan := make(chan peer.AddrInfo)

	h.SetStreamHandler("/chunk", func(s network.Stream) {
		wg.Add(1)
		go HandleStream(s, h, peerDataChan, &wg, *nodeID)
	})

	h.SetStreamHandler("/ready", func(s network.Stream) {
		wg.Add(1)
		go HandleReadyStream(s, h, &wg)
	})

	go DiscoveryHandler(h, peerChan)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		fmt.Println("\nReceived termination signal, shutting down...")
		cancel()
		h.Close()
		wg.Wait()
		os.Exit(0)
	}()

	fmt.Printf("Node %s is listening...\n", *nodeID)

	go func() {
		for pi := range peerChan {
			if err := h.Connect(ctx, pi); err != nil {
				fmt.Println("Error connecting to peer:", err)
				continue
			}

			fmt.Printf("Node %s connected to %s\n", *nodeID, pi.ID.String())
			connectedPeers = append(connectedPeers, pi)

			if *nodeID == "Node1" && len(connectedPeers) >= expectedChunks {
				originalData := "HelloLibP2P"
				chunks := DivideIntoChunks(originalData, chunkSize)
				nodeData := NodeData{originalData: originalData, chunks: chunks, received: make(map[int]string)}
				receivedChunks.Store(*nodeID, &nodeData)

				// Disperse the chunks to all connected peers
				for i, chunk := range chunks {
					SendChunk(ctx, h, connectedPeers[i], i, chunk)
				}
				break
			}
		}
	}()

	go func() {
		for pi := range peerDataChan {
			receivedChunk, ok := receivedChunks.Load(pi.ID.String())
			if !ok {
				fmt.Printf("Warning: No chunk found for peer %s\n", pi.ID.String())
				continue
			}

			nodeData := receivedChunk.(*NodeData)
			for index, chunk := range nodeData.received {
				for _, peerInfo := range connectedPeers {
					if peerInfo.ID != pi.ID && peerInfo.ID.String() != *nodeID && peerInfo.ID != node1ID {
						if _, ok := sentChunks.Load(peerInfo.ID.String() + chunk); !ok {
							SendChunk(ctx, h, peerInfo, index, chunk)
							sentChunks.Store(peerInfo.ID.String()+chunk, struct{}{})
						}
					}
				}
			}
		}
	}()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			if strings.TrimSpace(text) != "show" {
				PrintReceivedChunks()
			}
		}
	}()

	select {}
}

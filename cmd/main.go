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
	"github.com/xm0onh/AVID-go/handlers"
	"github.com/xm0onh/AVID-go/rs"
	"github.com/xm0onh/AVID-go/shared"
)

const (
	rendezvousString = "libp2p-mdns"
	dataShards       = 2
	parityShards     = 1
	expectedChunks   = dataShards + parityShards
)

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
		go handlers.HandleStream(s, h, peerDataChan, &wg, *nodeID)
	})

	h.SetStreamHandler("/ready", func(s network.Stream) {
		wg.Add(1)
		go handlers.HandleReadyStream(s, h, &wg)
	})

	go handlers.DiscoveryHandler(h, peerChan, rendezvousString)

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
			shared.ConnectedPeers = append(shared.ConnectedPeers, pi)

			if *nodeID == "Node1" && len(shared.ConnectedPeers) >= shared.ExpectedChunks {
				originalData := "HelloLibP2P"
				shards, err := rs.RSEncode(originalData)
				if err != nil {
					fmt.Printf("Node %s failed to encode data: %v\n", *nodeID, err)
					return
				}
				nodeData := shared.NodeData{OriginalData: originalData, Chunks: shards, Received: make(map[int][]byte)}
				shared.ReceivedChunks.Store(*nodeID, &nodeData)

				for i, shard := range shards[:shared.ExpectedChunks] {
					handlers.SendChunk(ctx, h, shared.ConnectedPeers[i], i, shard)
				}
				break
			}
		}
	}()

	go func() {
		for pi := range peerDataChan {
			receivedChunk, ok := shared.ReceivedChunks.Load(pi.ID.String())
			fmt.Println("Received chunk from", pi.ID.String())
			if !ok {
				fmt.Printf("Warning: No chunk found for peer %s\n", pi.ID.String())
				continue
			}

			nodeData := receivedChunk.(*shared.NodeData)
			for index, chunk := range nodeData.Received {
				receivedFromKey := fmt.Sprintf("%s-%d", pi.ID.String(), index)
				origSender, senderOk := shared.ReceivedFrom.Load(receivedFromKey)

				if senderOk && origSender.(string) == shared.Node1ID.String() {
					for _, peerInfo := range shared.ConnectedPeers {
						if peerInfo.ID != pi.ID && peerInfo.ID.String() != *nodeID && peerInfo.ID != shared.Node1ID {
							chunkKey := fmt.Sprintf("%s-%d", peerInfo.ID.String(), index)
							if _, ok := shared.SentChunks.Load(chunkKey); !ok {
								fmt.Printf("Sending chunk %d to %s\n", index, peerInfo.ID.String())
								handlers.SendChunk(context.Background(), h, peerInfo, index, chunk)
								shared.SentChunks.Store(chunkKey, struct{}{})
							} else {
								fmt.Printf("Chunk %d already sent to %s\n", index, peerInfo.ID.String())
							}
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
			if strings.TrimSpace(text) == "show" {
				handlers.PrintReceivedChunks(*nodeID)
			}
		}
	}()

	select {}
}

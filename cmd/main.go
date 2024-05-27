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
	"time"

	// "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	BT "github.com/xm0onh/AVID-go/bootstrap"
	"github.com/xm0onh/AVID-go/config"
	"github.com/xm0onh/AVID-go/handlers"
	lt "github.com/xm0onh/AVID-go/lt"
	"github.com/xm0onh/AVID-go/rs"
)

var broadcastSignal = make(chan struct{})
var log = logrus.New()

//	func init() {
//		log.SetLogLevel("*", "debug")
//	}
func main() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	nodeID := flag.String("node", "", "Node ID")
	bootstrap := flag.Bool("bootstrap", false, "Start as bootstrap node")
	port := flag.Int("port", 0, "Port to listen on")
	ip := flag.String("ip", "127.0.0.1", "IP address to listen on")
	codingMethod := flag.String("coding", "RS", "Coding method (RS, LT)")

	flag.Parse()

	if *codingMethod == "LT" {
		config.CodingMethod = "LT"
	} else if *codingMethod == "RS"{
		config.CodingMethod = "RS"
	} else {
		fmt.Println("Invalid coding method. Please specify either 'RS' or 'LT'.")
		os.Exit(1)
	}
	if *nodeID == "" {
		fmt.Println("Please specify node ID with -node flag.")
		os.Exit(1)
	}

	if *bootstrap {
		BT.StartBootstrapNode(*port, *ip)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
	if err != nil {
		fmt.Printf("Error generating key pair: %v\n", err)
		os.Exit(1)
	}
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", *ip, *port))
	if err != nil {
		fmt.Printf("Error creating multiaddr: %v\n", err)
		os.Exit(1)
	}
	// h, err := libp2p.New(libp2p.Identity(priv), libp2p.ListenAddrs(addr),
	// 	libp2p.DefaultTransports, libp2p.DefaultMuxers, libp2p.DefaultSecurity)

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrs(addr),
		libp2p.Security(tls.ID, tls.New),
		libp2p.Security(noise.ID, noise.New))

	if err != nil {
		fmt.Printf("Error creating libp2p host: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Node %s is listening on %s:%d with Peer ID %s...\n", *nodeID, *ip, *port, h.ID())

	// Write Node1's Peer ID as dispersal node to a file
	if *nodeID == "Node1" {
		file, err := os.Create("DispersalNode.txt")
		if err != nil {
			fmt.Printf("Error creating file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		_, err = file.WriteString(h.ID().String())
		if err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			os.Exit(1)
		}
	}

	// Read Node1's Peer ID from the file
	if *nodeID != "Node1" {
		file, err := os.Open("DispersalNode.txt")
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		if scanner.Scan() {
			node1PeerID, err := peer.Decode(scanner.Text())
			if err != nil {
				fmt.Printf("Error decoding peer ID: %v\n", err)
				os.Exit(1)
			}
			config.Node1ID = node1PeerID
		}
	}

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

	go handlers.DiscoveryHandler(h, peerChan)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		fmt.Println("\nReceived termination signal, shutting down...")
		cancel()
		h.Close()
		wg.Wait()
		os.Exit(1)
	}()

	go func() {
		for pi := range peerChan {

			if err := h.Connect(ctx, pi); err != nil {
				fmt.Printf("Error connecting to peer %s (retry %d): %v\n", pi.ID.String(), 0, err)
				for i := 1; i <= 10; i++ {
					time.Sleep(5 * time.Second)
					if err := h.Connect(ctx, pi); err == nil {
						fmt.Println("Finally Succeed!!")
						fmt.Printf("Node %s connected to %s\n", *nodeID, pi.ID.String())
						break
					} else {
						// fmt.Printf("Error connecting to peer %s (retry %d): %v\n", pi.ID.String(), i)
						fmt.Printf("Node %s attempting to connect to peer %s at %v\n", *nodeID, pi.ID.String(), pi.Addrs)
					}
				}

			}

			// fmt.Printf("Node %s connected to %s\n", *nodeID, pi.ID.String())
			fmt.Printf("Node %s connected to peer %s at %v\n", *nodeID, pi.ID.String(), pi.Addrs)
			config.ConnectedPeers = append(config.ConnectedPeers, pi)

			if *nodeID == "Node1" && len(config.ConnectedPeers) > config.Nodes-1 {
				fmt.Println("Node 1 is ready to broadcast chunks. Type 'start' to begin broadcasting.")
				<-broadcastSignal
				config.StartTime = time.Now()
				fmt.Println("Broadcasting chunks...")

				originalFilePath := "test.txt"
				originalData, err := os.ReadFile(originalFilePath)
				if err != nil {
					fmt.Printf("Node %s failed to read original data file: %v\n", *nodeID, err)
					return
				}

				var chunks [][]byte
				var originalLength int

				if *codingMethod == "RS" {
					chunks, err = rs.RSEncode(string(originalData))
					if err != nil {
						fmt.Printf("Node %s failed to encode data: %v\n", *nodeID, err)
						return
					}
				} else if *codingMethod == "LT" {
					fmt.Println("LT encoding")
					chunks, err = lt.LTEncode(string(originalData))
					if err != nil {
						fmt.Printf("Node %s failed to encode data: %v\n", *nodeID, err)
						return
					}
					originalLength = len(string(originalData))
					log.WithFields(logrus.Fields{"Original Length": originalLength}).Info("Org Size")
				} else {
					panic("Invalid coding method")
				}

				nodeData := config.NodeData{OriginalData: string(originalData), Chunks: chunks, Received: make(map[int][]byte)}
				config.ReceivedChunks.Store(*nodeID, &nodeData)
				fmt.Println("Leng of chunks", len(chunks))
				for i, chunk := range chunks {
					if *codingMethod == "LT" {
						fmt.Println("LT code broadcasting")
						
							handlers.SendChunkWithOriginalLength(ctx, h, config.ConnectedPeers[i], i, chunk, originalLength)
						
					} else {
						handlers.SendChunk(ctx, h, config.ConnectedPeers[i], i, chunk)
					}
				}
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for pi := range peerDataChan {
			receivedChunk, ok := config.ReceivedChunks.Load(pi.ID.String())
			fmt.Println("Received chunk from", pi.ID.String())
			if !ok {
				fmt.Printf("Warning: No chunk found for peer %s\n", pi.ID.String())
				continue
			}

			nodeData := receivedChunk.(*config.NodeData)
			for index, chunk := range nodeData.Received {
				receivedFromKey := fmt.Sprintf("%s-%d", pi.ID.String(), index)
				origSender, senderOk := config.ReceivedFrom.Load(receivedFromKey)

				if senderOk && origSender.(string) == config.Node1ID.String() {
					for _, peerInfo := range config.ConnectedPeers {
						if peerInfo.ID != pi.ID && peerInfo.ID.String() != *nodeID && peerInfo.ID != config.Node1ID {
							chunkKey := fmt.Sprintf("%s-%d", peerInfo.ID.String(), index)
							if _, ok := config.SentChunks.Load(chunkKey); !ok {
								fmt.Printf("Sending chunk %d to %s\n", index, peerInfo.ID.String())
								handlers.SendChunk(context.Background(), h, peerInfo, index, chunk)
								config.SentChunks.Store(chunkKey, struct{}{})
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
			fmt.Println("Received input:", text)
			fmt.Println("Node ID:", *nodeID)
			if strings.TrimSpace(text) == "show" {
				handlers.PrintReceivedChunks(*nodeID)
			} else if strings.TrimSpace(text) == "start" && *nodeID == "Node1" {
				fmt.Println("hi")
				close(broadcastSignal)
			} else if strings.TrimSpace(text) == "exit" {
				cancel()
				h.Close()
				wg.Wait()
				os.Exit(0)
			} else if strings.TrimSpace(text) == "con" {
				fmt.Println("Number of Connected Peers:", len(config.ConnectedPeers))
				fmt.Println("Connected Peers:", config.ConnectedPeers)
			}
		}
	}()

	select {}
}

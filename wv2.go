package main

// import (
// 	"bufio"
// 	"context"
// 	"crypto/rand"
// 	"encoding/binary"
// 	"flag"
// 	"fmt"
// 	"os"
// 	"os/signal"
// 	"sync"
// 	"syscall"

// 	libp2p "github.com/libp2p/go-libp2p"
// 	crypto "github.com/libp2p/go-libp2p/core/crypto"
// 	"github.com/libp2p/go-libp2p/core/host"
// 	network "github.com/libp2p/go-libp2p/core/network"
// 	peer "github.com/libp2p/go-libp2p/core/peer"
// 	protocol "github.com/libp2p/go-libp2p/core/protocol"
// 	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
// )

// const rendezvousString = "libp2p-mdns"
// const chunkSize = 3 // Define the number of chunks

// func handleStream(s network.Stream, wg *sync.WaitGroup) {
// 	defer s.Close()
// 	defer wg.Done()

// 	fmt.Println("Received connection from:", s.Conn().RemotePeer())

// 	reader := bufio.NewReader(s)

// 	for {
// 		var length uint32
// 		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
// 			if err.Error() == "EOF" {
// 				break
// 			}
// 			fmt.Println("Error reading length from stream:", err)
// 			s.Reset()
// 			return
// 		}

// 		buf := make([]byte, length)
// 		if _, err := reader.Read(buf); err != nil {
// 			fmt.Println("Error reading from stream:", err)
// 			s.Reset()
// 			return
// 		}

// 		receivedData := string(buf)
// 		fmt.Printf("Received data: %s\n", receivedData)
// 	}
// }

// func discoveryHandler(h host.Host, peerChan chan peer.AddrInfo) {
// 	mdnsService := mdns.NewMdnsService(h, rendezvousString, &discoveryNotifee{peerChan: peerChan})
// 	if err := mdnsService.Start(); err != nil {
// 		fmt.Println("Error starting mDNS service:", err)
// 		os.Exit(1)
// 	}
// }

// type discoveryNotifee struct {
// 	peerChan chan peer.AddrInfo
// }

// func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
// 	n.peerChan <- pi
// }

// func main() {
// 	nodeID := flag.String("node", "", "Node ID")
// 	flag.Parse()

// 	if *nodeID == "" {
// 		fmt.Println("Please specify node ID with -node flag.")
// 		os.Exit(1)
// 	}

// 	ctx, cancel := context.WithCancel(context.Background())
// 	priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
// 	h, _ := libp2p.New(libp2p.Identity(priv))

// 	var wg sync.WaitGroup
// 	h.SetStreamHandler(protocol.ID("/chunk"), func(s network.Stream) {
// 		wg.Add(1)
// 		go handleStream(s, &wg)
// 	})

// 	peerChan := make(chan peer.AddrInfo)
// 	go discoveryHandler(h, peerChan)

// 	go func() {
// 		c := make(chan os.Signal, 1)
// 		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
// 		<-c
// 		fmt.Println("\nReceived termination signal, shutting down...")
// 		cancel()
// 		h.Close()
// 		wg.Wait()
// 		os.Exit(0)
// 	}()

// 	fmt.Printf("Node %s is listening...\n", *nodeID)

// 	var connectedPeers []peer.AddrInfo

// 	go func() {
// 		for pi := range peerChan {
// 			if err := h.Connect(ctx, pi); err != nil {
// 				fmt.Println("Error connecting to peer:", err)
// 				continue
// 			}

// 			fmt.Printf("Node %s connected to %s\n", *nodeID, pi.ID.String())
// 			connectedPeers = append(connectedPeers, pi)

// 			if *nodeID == "Node1" && len(connectedPeers) >= 2 {
// 				originalData := "HelloLibP2P" 
// 				chunks := divideIntoChunks(originalData, chunkSize)

// 				for i, pi := range connectedPeers {
// 					sendChunk(ctx, h, pi, chunks[i+1]) 
// 				}
				
// 				break
// 			}
// 		}
// 	}()

// 	select {}
// }

// func divideIntoChunks(data string, numChunks int) []string {
// 	chunkLength := len(data) / numChunks
// 	chunks := make([]string, numChunks)

// 	for i := 0; i < numChunks; i++ {
// 		start := i * chunkLength
// 		end := start + chunkLength
// 		if i == numChunks-1 {
// 			end = len(data) // Make sure the last chunk includes any remaining characters
// 		}
// 		chunks[i] = data[start:end]
// 	}
// 	return chunks
// }

// func sendChunk(ctx context.Context, h host.Host, pi peer.AddrInfo, chunk string) {
// 	s, err := h.NewStream(ctx, pi.ID, protocol.ID("/chunk"))
// 	if err != nil {
// 		fmt.Println("Error creating stream:", err)
// 		return
// 	}
// 	defer s.Close()

// 	if err := binary.Write(s, binary.LittleEndian, uint32(len(chunk))); err != nil {
// 		fmt.Println("Error writing length to stream:", err)
// 		s.Reset()
// 		return
// 	}

// 	if _, err := s.Write([]byte(chunk)); err != nil {
// 		fmt.Println("Error writing to stream:", err)
// 		s.Reset()
// 		return
// 	}

// 	fmt.Printf("Sent chunk to %s: %s\n", pi.ID.String(), chunk)
// }

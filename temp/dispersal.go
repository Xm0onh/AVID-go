package main

// import (
// 	"context"
// 	"crypto/sha256"
// 	"fmt"
// 	"log"
// 	"sync"

// 	"github.com/libp2p/go-libp2p/core/host"
// 	"github.com/libp2p/go-libp2p/core/network"
// )

// var (
// 	storedChunks   = make(map[string][]byte)
// 	storedChunksMu sync.Mutex
// )

// func DisperseData(ctx context.Context, h host.Host, data []byte, method string, n, k int) error {
// 	var chunks [][]byte
// 	var err error

// 	if method == "RS" {
// 		chunks, _, err = EncodeRS(data, n, k)
// 	} else {
// 		return fmt.Errorf("unknown method: %s", method)
// 	}

// 	if err != nil {
// 		return err
// 	}

// 	for i, chunk := range chunks {
// 		hash := sha256.Sum256(chunk)
// 		hashStr := fmt.Sprintf("%x", hash)
// 		msg := append([]byte("disperse "), hash[:]...)
// 		msg = append(msg, chunk...)

// 		storedChunksMu.Lock()
// 		storedChunks[hashStr] = chunk
// 		storedChunksMu.Unlock()

// 		for _, peerID := range h.Network().Peers() {
// 			fmt.Printf("Sending handshake to peer %s\n", peerID.String())
// 			s, err := h.NewStream(ctx, peerID, protocolID)
// 			if err != nil {
// 				log.Println("Error creating stream:", err)
// 				continue
// 			}
// 			_, err = s.Write([]byte("handshake "))
// 			if err != nil {
// 				log.Println("Error sending handshake:", err)
// 			}
// 			s.Close()
// 		}

// 		for _, peerID := range h.Network().Peers() {
// 			fmt.Printf("Sending chunk %d to peer %s\n", i, peerID.String())
// 			s, err := h.NewStream(ctx, peerID, protocolID)
// 			if err != nil {
// 				log.Println("Error creating stream:", err)
// 				continue
// 			}
// 			_, err = s.Write(msg)
// 			if err != nil {
// 				log.Println("Error sending chunk:", err)
// 			}
// 			s.Close()
// 		}
// 	}

// 	return nil
// }

// func HandleStream(s network.Stream) {
// 	defer s.Close()

// 	buf := make([]byte, 4096) // Increased buffer size to handle larger chunks
// 	n, err := s.Read(buf)
// 	if err != nil {
// 		log.Println("Error reading from stream:", err)
// 		return
// 	}

// 	if n < 9 { // Ensure command and data are present
// 		log.Println("Received data is too short")
// 		return
// 	}

// 	command := string(buf[:8])
// 	switch command {
// 	case "handshak":
// 		handleHandshakeCommand(buf[9:n], s)
// 	case "disperse":
// 		handleDisperseCommand(buf[9:n], s)
// 	case "retrieve":
// 		handleRetrieveCommand(buf[9:n], s)
// 	default:
// 		log.Println("Unknown command received")
// 	}
// }

// func handleHandshakeCommand(data []byte, s network.Stream) {
// 	fmt.Printf("Handshake received from %s\n", s.Conn().RemotePeer())
// 	_, err := s.Write([]byte("handshake ack"))
// 	if err != nil {
// 		log.Println("Error sending handshake ack:", err)
// 	}
// }

// func handleDisperseCommand(data []byte, s network.Stream) {
// 	if len(data) < sha256.Size {
// 		log.Println("Disperse command data is too short")
// 		return
// 	}

// 	hash := data[:sha256.Size]
// 	chunk := data[sha256.Size:]

// 	if !equalHash(hash, sha256.Sum256(chunk)) {
// 		log.Println("Hash mismatch for received chunk")
// 		return
// 	}

// 	fmt.Printf("Received chunk from %s: %x\n", s.Conn().RemotePeer(), hash)
// 	hashStr := fmt.Sprintf("%x", hash)
// 	storedChunksMu.Lock()
// 	storedChunks[hashStr] = chunk
// 	storedChunksMu.Unlock()
// }

// func handleRetrieveCommand(data []byte, s network.Stream) {
// 	// Send the stored chunks back to the requesting peer
// 	storedChunksMu.Lock()
// 	defer storedChunksMu.Unlock()

// 	for hashStr, chunk := range storedChunks {
// 		fmt.Printf("Sending chunk %x\n", hashStr)
// 		hash := []byte(hashStr)
// 		msg := append(hash, chunk...)
// 		_, err := s.Write(msg)
// 		if err != nil {
// 			log.Println("Error sending chunk:", err)
// 		}
// 	}
// }

// func equalHash(hash []byte, sum [sha256.Size]byte) bool {
// 	for i := 0; i < sha256.Size; i++ {
// 		if hash[i] != sum[i] {
// 			return false
// 		}
// 	}
// 	return true
// }

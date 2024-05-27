package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/xm0onh/AVID-go/config"
	lt "github.com/xm0onh/AVID-go/lt"
	"github.com/xm0onh/AVID-go/rs"
)

var log = logrus.New()

func HandleStream(s network.Stream, h host.Host, peerChan chan peer.AddrInfo, wg *sync.WaitGroup, nodeID string) {
	defer s.Close()
	defer wg.Done()

	fmt.Println("Received connection from:", s.Conn().RemotePeer())

	reader := bufio.NewReader(s)

	config.NodeMutex.Lock()
	if config.Node1ID == "" {
		config.Node1ID = s.Conn().RemotePeer()
		fmt.Printf("Node %s set Node1ID to %s\n", nodeID, config.Node1ID.String())
	}
	config.NodeMutex.Unlock()

	for {
		// If this is the first chunk, read the original length
		if config.Counter == 0 {
			var originalLength int32
			if err := binary.Read(reader, binary.LittleEndian, &originalLength); err != nil {
				fmt.Println("Error reading original length from stream:", err)
				s.Reset()
				return
			}
			config.OriginalLength = int(originalLength)
			fmt.Printf("Received original length: %d\n", config.OriginalLength)
		}

		// Read the length of the incoming chunk data
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading length from stream:", err)
			s.Reset()
			return
		}

		// Read the actual chunk data
		buf := make([]byte, length)
		if _, err := reader.Read(buf); err != nil {
			fmt.Println("Error reading from stream:", err)
			s.Reset()
			return
		}

		// Read the chunk index
		var chunkIndex int32
		if err := binary.Read(reader, binary.LittleEndian, &chunkIndex); err != nil {
			fmt.Println("Error reading chunk index from stream:", err)
			s.Reset()
			return
		}

		// At this point, buf only contains the chunk data
		receivedData := buf
		// fmt.Printf("Received data chunk %d: %s\n", chunkIndex, receivedData)

		// Store the received chunk for later processing
		receivedFromKey := fmt.Sprintf("%s-%d", s.Conn().RemotePeer().String(), chunkIndex)
		fmt.Printf("Storing origin %s for chunk\n", s.Conn().RemotePeer().String())
		config.ReceivedFrom.Store(receivedFromKey, s.Conn().RemotePeer().String())

		// Call the function to store the received chunk
		StoreReceivedChunk(s.Conn().RemotePeer().String(), int(chunkIndex), receivedData, h, peerChan)

		// Notify that a chunk has been received
		peerChan <- peer.AddrInfo{ID: s.Conn().RemotePeer()}
	}
}

func HandleReadyStream(s network.Stream, h host.Host, wg *sync.WaitGroup) {
	defer s.Close()
	defer wg.Done()

	config.ReadyCounter++
	reader := bufio.NewReader(s)
	buf, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from stream:", err)
		s.Reset()
		return
	}

	peerID := strings.TrimSpace(buf)
	fmt.Printf("Ready message received for peer %s\n", peerID)
	if config.ReadyCounter == config.ExpectedChunks {
		fmt.Println("All nodes are ready")
		fmt.Println("Total time", time.Since(config.StartTime).Milliseconds(), "ms")
	}
}

func SendChunkWithOriginalLength(ctx context.Context, h host.Host, pi peer.AddrInfo, chunkIndex int, chunk []byte, originalLength int) {
	s, err := h.NewStream(ctx, pi.ID, protocol.ID("/chunk"))
	if err != nil {
		fmt.Println("Error creating stream:", err)
		return
	}
	defer s.Close()

	// First, write the original length
	if err := binary.Write(s, binary.LittleEndian, int32(originalLength)); err != nil {
		fmt.Println("Error writing original length to stream:", err)
		s.Reset()
		return
	}

	// Next, write the length of the chunk
	if err := binary.Write(s, binary.LittleEndian, uint32(len(chunk))); err != nil {
		fmt.Println("Error writing length to stream:", err)
		s.Reset()
		return
	}

	// Write the chunk data
	if _, err := s.Write(chunk); err != nil {
		fmt.Println("Error writing to stream:", err)
		s.Reset()
		return
	}

	// Finally, write the chunk index
	if err := binary.Write(s, binary.LittleEndian, int32(chunkIndex)); err != nil {
		fmt.Println("Error writing chunk index to stream:", err)
		s.Reset()
		return
	}

	fmt.Printf("Sent chunk with original length to %s: %s\n", pi.ID.String(), chunk)
}

func SendChunk(ctx context.Context, h host.Host, pi peer.AddrInfo, chunkIndex int, chunk []byte) {
	s, err := h.NewStream(ctx, pi.ID, protocol.ID("/chunk"))
	if err != nil {
		fmt.Println("Error creating stream:", err)
		return
	}
	defer s.Close()

	if err := binary.Write(s, binary.LittleEndian, uint32(len(chunk))); err != nil {
		fmt.Println("Error writing length to stream:", err)
		s.Reset()
		return
	}

	if _, err := s.Write(chunk); err != nil {
		fmt.Println("Error writing to stream:", err)
		s.Reset()
		return
	}

	if err := binary.Write(s, binary.LittleEndian, int32(chunkIndex)); err != nil {
		fmt.Println("Error writing chunk index to stream:", err)
		s.Reset()
		return
	}

	fmt.Printf("Sent chunk to %s: %s\n", pi.ID.String(), chunk)
}

func SendReady(ctx context.Context, h host.Host, pi peer.AddrInfo, peerID string) {
	s, err := h.NewStream(ctx, pi.ID, protocol.ID("/ready"))
	if err != nil {
		fmt.Println("Error creating stream:", err)
		return
	}
	defer s.Close()

	writer := bufio.NewWriter(s)
	if _, err := writer.WriteString(peerID + "\n"); err != nil {
		fmt.Println("Error writing to stream:", err)
		s.Reset()
		return
	}
	if err := writer.Flush(); err != nil {
		fmt.Println("Error flushing stream:", err)
		s.Reset()
		return
	}

	fmt.Printf("Sent ready message to %s for peer %s\n", pi.ID.String(), peerID)
}

func StoreReceivedChunk(nodeID string, chunkIndex int, chunk []byte, h host.Host, peerChan chan peer.AddrInfo) {
	config.NodeMutex.Lock()
	defer config.NodeMutex.Unlock()

	config.Counter++
	data, ok := config.ReceivedChunks.Load(nodeID)
	if !ok {
		fmt.Println("NodeID not found in receivedChunks", nodeID)
		data = &config.NodeData{Received: make(map[int][]byte)}
		config.ReceivedChunks.Store(nodeID, data)
	}
	nodeData := data.(*config.NodeData)
	config.ChunksRecByNode[chunkIndex] = chunk
	fmt.Println("counter", config.Counter)
	// fmt.Println("chunksRecByNode", config.ChunksRecByNode)
	if existingChunk, exists := nodeData.Received[chunkIndex]; exists && bytes.Equal(existingChunk, chunk) {
		fmt.Printf("Node %s already has chunk %d: %s\n", nodeID, chunkIndex, chunk)
		return
	}

	nodeData.Received[chunkIndex] = chunk
	// fmt.Printf("nodeid %s data %v\n", nodeID, nodeData.Received)
	fmt.Printf("Node %s received chunk %d: %s\n", nodeID, chunkIndex, chunk)
	fmt.Println("Length of received chunks:", len(nodeData.Received))

	if config.Counter == config.ExpectedChunks {
		log.WithFields(logrus.Fields{"nodeID": nodeID}).Info("Node complete received data")

		var decodedData string
		var err error
		fmt.Println("Coding method", config.CodingMethod)
		if config.CodingMethod == "LT" {
			decodedData, err = lt.LTDecode(config.ChunksRecByNode[:config.ExpectedChunks], config.OriginalLength)
			fmt.Println("decodedData", decodedData)
		} else if config.CodingMethod == "RS" {
			decodedData, err = rs.RSDecode(config.ChunksRecByNode[:config.ExpectedChunks])
		}

		if err != nil {
			fmt.Printf("Node %s failed to decode data: %v\n", nodeID, err)
			return
		}
		// Store the reconstructed data into a file
		outputFilePath := fmt.Sprintf("output/%s_out.txt", nodeID)
		if err := os.WriteFile(outputFilePath, []byte(decodedData), 0644); err != nil {
			fmt.Printf("Node %s failed to write reconstructed data to file: %v\n", nodeID, err)
			return
		}
		// fmt.Printf("Node %s reconstructed data: %s\n", nodeID, decodedData)
		log.WithFields(logrus.Fields{"nodeID": nodeID}).Info("Node reconstructed data")

		for _, peerInfo := range config.ConnectedPeers {
			if peerInfo.ID.String() != nodeID {
				readyKey := fmt.Sprintf("%s-ready", peerInfo.ID.String())
				if _, ok := config.SentChunks.Load(readyKey); !ok {
					SendReady(context.Background(), h, peerInfo, nodeID)
					config.SentChunks.Store(readyKey, struct{}{})
				}
			}
		}
	}
}

func PrintReceivedChunks(nodeID string) {
	fmt.Printf("Node %s printing its received chunks:\n", nodeID)
	value, ok := config.ReceivedChunks.Load(nodeID)
	if !ok {
		fmt.Printf("Node %s has not received any chunks\n", nodeID)
		return
	}

	nodeData := value.(*config.NodeData)
	if len(nodeData.Received) == config.ExpectedChunks {
		var completeData strings.Builder

		for i := 0; i < config.ExpectedChunks; i++ {
			chunk, exists := nodeData.Received[i]
			if exists {
				completeData.WriteString(string(chunk))
			} else {
				fmt.Printf("Node %s is missing chunk %d\n", nodeID, i)
				return
			}
		}

		fmt.Printf("Node %s reconstructed data: %s\n", nodeID, completeData.String())
	} else {
		fmt.Printf("Node %s has incomplete data\n", nodeID)
	}
}

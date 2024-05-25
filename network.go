package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type NodeData struct {
	originalData string
	chunks       []string
	received     map[int]string
}

func HandleStream(s network.Stream, h host.Host, peerChan chan peer.AddrInfo, wg *sync.WaitGroup, nodeID string) {
	defer s.Close()
	defer wg.Done()

	fmt.Println("Received connection from:", s.Conn().RemotePeer())

	reader := bufio.NewReader(s)

	if node1ID == "" {
		node1ID = s.Conn().RemotePeer()
		fmt.Printf("Node %s set Node1ID to %s\n", nodeID, node1ID.String())
	}

	for {
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading length from stream:", err)
			s.Reset()
			return
		}

		buf := make([]byte, length)
		if _, err := reader.Read(buf); err != nil {
			fmt.Println("Error reading from stream:", err)
			s.Reset()
			return
		}

		var chunkIndex int32
		if err := binary.Read(reader, binary.LittleEndian, &chunkIndex); err != nil {
			fmt.Println("Error reading chunk index from stream:", err)
			s.Reset()
			return
		}

		receivedData := string(buf)
		StoreReceivedChunk(s.Conn().RemotePeer().String(), int(chunkIndex), receivedData, h, peerChan)
		fmt.Printf("Received data: %s\n", receivedData)

		peerChan <- peer.AddrInfo{ID: s.Conn().RemotePeer()}
	}
}

func HandleReadyStream(s network.Stream, h host.Host, wg *sync.WaitGroup) {
	defer s.Close()
	defer wg.Done()

	fmt.Println("Received ready message from:", s.Conn().RemotePeer())

	reader := bufio.NewReader(s)
	buf, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from stream:", err)
		s.Reset()
		return
	}

	peerID := strings.TrimSpace(buf)
	fmt.Printf("Ready message received for peer %s\n", peerID)

}

func SendChunk(ctx context.Context, h host.Host, pi peer.AddrInfo, chunkIndex int, chunk string) {
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

	if _, err := s.Write([]byte(chunk)); err != nil {
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

func StoreReceivedChunk(peerID string, chunkIndex int, chunk string, h host.Host, peerChan chan peer.AddrInfo) {
	nodeMutex.Lock()
	defer nodeMutex.Unlock()

	data, ok := receivedChunks.Load(peerID)
	if !ok {
		data = &NodeData{received: make(map[int]string)}
		receivedChunks.Store(peerID, data)
	}
	nodeData := data.(*NodeData)
	nodeData.received[chunkIndex] = chunk

	if len(nodeData.received) == expectedChunks {
		var completeData strings.Builder
		chunkIndices := make([]string, expectedChunks)
		for i := 0; i < expectedChunks; i++ {
			chunkIndices[i] = nodeData.received[i]
		}
		for _, chunk := range chunkIndices {
			completeData.WriteString(chunk)
		}
		fmt.Printf("Node %s complete received data: %s\n", peerID, completeData.String())

		// Broadcast ready message
		for _, peerInfo := range connectedPeers {
			if _, ok := sentChunks.Load(peerInfo.ID.String() + "ready"); !ok {
				SendReady(context.Background(), h, peerInfo, peerID)
				sentChunks.Store(peerInfo.ID.String()+"ready", struct{}{})
			}
		}
	}
}

func PrintReceivedChunks() {
	fmt.Println("Printing all received chunks:")
	receivedChunks.Range(func(key, value interface{}) bool {
		peerID := key.(string)
		nodeData := value.(*NodeData)

		if len(nodeData.received) == expectedChunks {
			var completeData strings.Builder
			chunkIndices := make([]string, expectedChunks)
			for i := 0; i < expectedChunks; i++ {
				chunkIndices[i] = nodeData.received[i]
			}
			for _, chunk := range chunkIndices {
				completeData.WriteString(chunk)
			}
			fmt.Printf("Node %s reconstructed data: %s\n", peerID, completeData.String())
		} else {
			fmt.Printf("Node %s has incomplete data\n", peerID)
		}

		return true
	})
}

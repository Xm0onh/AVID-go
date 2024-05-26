package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/xm0onh/AVID-go/rs"
)

type NodeData struct {
	originalData string
	chunks       [][]byte
	received     map[int][]byte
}

func HandleStream(s network.Stream, h host.Host, peerChan chan peer.AddrInfo, wg *sync.WaitGroup, nodeID string, node1ID *peer.ID, receivedFrom, receivedChunks, sentChunks *sync.Map, nodeMutex *sync.Mutex, counter *int, chunksRecByNode [][]byte, readyCounter *int, connectedPeers []peer.AddrInfo, expectedChunks int) {
	defer s.Close()
	defer wg.Done()

	fmt.Println("Received connection from:", s.Conn().RemotePeer())

	reader := bufio.NewReader(s)

	nodeMutex.Lock()
	if *node1ID == "" {
		*node1ID = s.Conn().RemotePeer()
		fmt.Printf("Node %s set Node1ID to %s\n", nodeID, node1ID.String())
	}
	nodeMutex.Unlock()

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

		receivedData := buf
		fmt.Printf("Received data: %s\n", receivedData)

		receivedFromKey := fmt.Sprintf("%s-%d", s.Conn().RemotePeer().String(), chunkIndex)
		fmt.Printf("Storing origin %s for chunk %d\n", s.Conn().RemotePeer().String(), chunkIndex)
		receivedFrom.Store(receivedFromKey, s.Conn().RemotePeer().String())

		StoreReceivedChunk(s.Conn().RemotePeer().String(), int(chunkIndex), receivedData, h, peerChan, nodeMutex, counter, chunksRecByNode, connectedPeers, sentChunks, receivedChunks, expectedChunks)

		peerChan <- peer.AddrInfo{ID: s.Conn().RemotePeer()}
	}
}

func HandleReadyStream(s network.Stream, h host.Host, wg *sync.WaitGroup, readyCounter *int, connectedPeers []peer.AddrInfo, expectedChunks int) {
	defer s.Close()
	defer wg.Done()

	(*readyCounter)++
	reader := bufio.NewReader(s)
	buf, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from stream:", err)
		s.Reset()
		return
	}

	peerID := strings.TrimSpace(buf)
	fmt.Printf("Ready message received for peer %s\n", peerID)
	if *readyCounter == expectedChunks {
		fmt.Println("All nodes are ready")
	}
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

func StoreReceivedChunk(nodeID string, chunkIndex int, chunk []byte, h host.Host, peerChan chan peer.AddrInfo, nodeMutex *sync.Mutex, counter *int, chunksRecByNode [][]byte, connectedPeers []peer.AddrInfo, sentChunks *sync.Map, receivedChunks *sync.Map, expectedChunks int) {
	nodeMutex.Lock()
	defer nodeMutex.Unlock()

	(*counter)++
	data, ok := receivedChunks.Load(nodeID)
	if !ok {
		fmt.Println("NodeID not found in receivedChunks", nodeID)
		data = &NodeData{received: make(map[int][]byte)}
		receivedChunks.Store(nodeID, data)
	}
	nodeData := data.(*NodeData)
	chunksRecByNode[chunkIndex] = chunk
	fmt.Println("counter", *counter)
	fmt.Println("chunksRecByNode", chunksRecByNode)
	if existingChunk, exists := nodeData.received[chunkIndex]; exists && bytes.Equal(existingChunk, chunk) {
		fmt.Printf("Node %s already has chunk %d: %s\n", nodeID, chunkIndex, chunk)
		return
	}

	nodeData.received[chunkIndex] = chunk
	fmt.Printf("nodeid %s data %v\n", nodeID, nodeData.received)
	fmt.Printf("Node %s received chunk %d: %s\n", nodeID, chunkIndex, chunk)
	fmt.Println("Length of received chunks:", len(nodeData.received))

	if *counter == expectedChunks {
		fmt.Printf("Node %s complete received data\n", nodeID)
		decodedData, err := rs.RSDecode(chunksRecByNode)
		if err != nil {
			fmt.Printf("Node %s failed to decode data: %v\n", nodeID, err)
			return
		}
		fmt.Printf("Node %s reconstructed data: %s\n", nodeID, decodedData)

		for _, peerInfo := range connectedPeers {
			if peerInfo.ID.String() != nodeID {
				readyKey := fmt.Sprintf("%s-ready", peerInfo.ID.String())
				if _, ok := sentChunks.Load(readyKey); !ok {
					SendReady(context.Background(), h, peerInfo, nodeID)
					sentChunks.Store(readyKey, struct{}{})
				}
			}
		}
	}
}

func PrintReceivedChunks(nodeID string, receivedChunks *sync.Map, expectedChunks int) {
	fmt.Printf("Node %s printing its received chunks:\n", nodeID)
	value, ok := receivedChunks.Load(nodeID)
	if !ok {
		fmt.Printf("Node %s has not received any chunks\n", nodeID)
		return
	}

	nodeData := value.(*NodeData)
	if len(nodeData.received) == expectedChunks {
		var completeData strings.Builder

		for i := 0; i < expectedChunks; i++ {
			chunk, exists := nodeData.received[i]
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

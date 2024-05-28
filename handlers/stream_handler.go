package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
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

var flag = true

const subChunkSize = 256 * 1024 // 256 KB
const maxRetries = 5

func init() {
	// Initialize the logger
	log.SetLevel(logrus.DebugLevel)
}

// Helper function to write sub-chunks to the stream
func writeSubChunks(stream network.Stream, chunk []byte) error {
	writer := bufio.NewWriterSize(stream, 4*subChunkSize) // Set buffer size
	for offset := 0; offset < len(chunk); offset += subChunkSize {
		end := offset + subChunkSize
		if end > len(chunk) {
			end = len(chunk)
		}
		subChunk := chunk[offset:end]
		// fmt.Printf("Writing sub-chunk of length %d\n", len(subChunk))
		if err := binary.Write(writer, binary.LittleEndian, uint32(len(subChunk))); err != nil {
			return err
		}
		if err := writeAll(writer, subChunk); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
		// fmt.Printf("Wrote sub-chunk from offset %d to %d\n", offset, end)
	}
	return nil
}

// Helper function to write all bytes
func writeAll(writer *bufio.Writer, data []byte) error {
	totalWritten := 0
	for totalWritten < len(data) {
		n, err := writer.Write(data[totalWritten:])
		if err != nil {
			return err
		}
		totalWritten += n
	}
	return nil
}

func readSubChunks(reader *bufio.Reader, length uint32) ([]byte, error) {
	var buffer bytes.Buffer
	for length > 0 {
		var subChunkLength uint32
		if err := binary.Read(reader, binary.LittleEndian, &subChunkLength); err != nil {
			return nil, fmt.Errorf("failed to read sub-chunk length: %w", err)
		}
		// fmt.Printf("Reading sub-chunk of length %d\n", subChunkLength)

		// Check if the subChunkLength is reasonable
		if subChunkLength > length || subChunkLength > subChunkSize {
			return nil, fmt.Errorf("invalid sub-chunk length: %d", subChunkLength)
		}

		subChunk := make([]byte, subChunkLength)
		if err := readAll(reader, subChunk); err != nil {
			return nil, err
		}

		buffer.Write(subChunk)
		length -= subChunkLength
		// fmt.Printf("Read sub-chunk of length %d, remaining length %d\n", subChunkLength, length)
	}
	return buffer.Bytes(), nil
}

// Helper function to read all bytes
func readAll(reader *bufio.Reader, data []byte) error {
	totalRead := 0
	for totalRead < len(data) {
		n, err := reader.Read(data[totalRead:])
		if err != nil {
			// Retry mechanism
			retries := 0
			for retries < maxRetries && err != nil {
				time.Sleep(time.Second * time.Duration(retries+1))
				n, err = reader.Read(data[totalRead:])
				retries++
			}
			if err != nil {
				return fmt.Errorf("failed to read data after %d retries: %w", retries, err)
			}
		}
		totalRead += n
	}
	return nil
}

func HandleStream(s network.Stream, h host.Host, peerChan chan peer.AddrInfo, wg *sync.WaitGroup, nodeID string) {
	defer s.Close()
	defer wg.Done()

	fmt.Println("Received connection from:", s.Conn().RemotePeer())

	reader := bufio.NewReaderSize(s, 4*subChunkSize) // Set buffer size

	config.NodeMutex.Lock()
	if config.Node1ID == "" {
		config.Node1ID = s.Conn().RemotePeer()
		fmt.Printf("Node %s set Node1ID to %s\n", nodeID, config.Node1ID.String())
	}
	config.NodeMutex.Unlock()

	for {
		var chunkIndex int32
		if err := binary.Read(reader, binary.LittleEndian, &chunkIndex); err != nil {
			if err.Error() == "EOF" {
				fmt.Println("Stream closed by sender") // Added logging for EOF
				break
			}
			fmt.Println("Error reading chunk index from stream:", err)
			s.Reset()
			return
		}

		fmt.Printf("Chunk index ------>: %d\n", chunkIndex)

		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error reading length from stream:", err)
			s.Reset()
			return
		}
		fmt.Println("Length of chunk: ", length)
		if length > 100*1024*1024 {
			fmt.Printf("Unreasonably large chunk length received: %d\n", length)
			s.Reset()
			return
		}

		buf, err := readSubChunks(reader, length)
		if err != nil {
			fmt.Println("Error reading sub-chunks from stream:", err)
			s.Reset()
			return
		}

		receivedData := buf

		receivedFromKey := fmt.Sprintf("%s-%d", s.Conn().RemotePeer().String(), chunkIndex)
		fmt.Printf("Storing origin %s for chunk %d\n", s.Conn().RemotePeer().String(), chunkIndex)
		config.ReceivedFrom.Store(receivedFromKey, s.Conn().RemotePeer().String())

		StoreReceivedChunk(s.Conn().RemotePeer().String(), int(chunkIndex), receivedData, h, peerChan)

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
	if config.ReadyCounter >= int((config.ExpectedChunks)*2/3+1) {
		fmt.Println("Nodes are ready")
		fmt.Println("Total time", time.Since(config.StartTime).Milliseconds(), "ms")
		// panic("Done")
	}
}

func SendChunk(ctx context.Context, h host.Host, pi peer.AddrInfo, chunkIndex int, chunk []byte) {
	if len(chunk) == 0 {
		fmt.Printf("Not sending empty chunk %d\n", chunkIndex)
		return
	}

	var retryCount int

	for retryCount = 0; retryCount < maxRetries; retryCount++ {
		streamCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		s, err := h.NewStream(streamCtx, pi.ID, protocol.ID("/chunk"))
		if err != nil {
			fmt.Println("Error creating stream:", err)
			time.Sleep(time.Duration(retryCount+1) * time.Second) // Exponential backoff
			continue
		}
		defer s.Close()

		fmt.Printf("Sending chunk index %d\n", chunkIndex)

		if err := binary.Write(s, binary.LittleEndian, int32(chunkIndex)); err != nil {
			fmt.Println("Error writing chunk index to stream:", err)
			s.Reset()
			time.Sleep(time.Duration(retryCount+1) * time.Second) // Exponential backoff
			continue
		}

		fmt.Printf("Sending chunk length %d\n", len(chunk))

		if err := binary.Write(s, binary.LittleEndian, uint32(len(chunk))); err != nil {
			fmt.Println("Error writing length to stream:", err)
			s.Reset()
			time.Sleep(time.Duration(retryCount+1) * time.Second) // Exponential backoff
			continue
		}

		if err := writeSubChunks(s, chunk); err != nil {
			fmt.Println("Error writing sub-chunks to stream:", err)
			s.Reset()
			time.Sleep(time.Duration(retryCount+1) * time.Second) // Exponential backoff
			continue
		}

		fmt.Printf("Sent chunk %d to %s\n", chunkIndex, pi.ID.String())
		break
	}

	if retryCount == maxRetries {
		fmt.Printf("Failed to send chunk %d to %s after %d attempts\n", chunkIndex, pi.ID.String(), maxRetries)
	}
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
	if flag {
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

		if chunkIndex >= len(config.ChunksRecByNode) {
			fmt.Printf("Received chunk index %d out of range, ignoring chunk\n", chunkIndex)
			return
		}

		config.ChunksRecByNode[chunkIndex] = chunk
		fmt.Println("counter", config.Counter)

		if existingChunk, exists := nodeData.Received[chunkIndex]; exists && bytes.Equal(existingChunk, chunk) {
			fmt.Printf("Node %s already has chunk %d: %s\n", nodeID, chunkIndex, chunk)
			return
		}

		nodeData.Received[chunkIndex] = chunk
		fmt.Printf("Node %s received chunk %d\n", nodeID, chunkIndex)
		fmt.Println("Length of received chunks:", len(nodeData.Received))

		if config.Counter == config.ExpectedChunks {
			log.WithFields(logrus.Fields{"nodeID": nodeID}).Info("Node complete received data")
			// var decodedData string
			var err error
			log.WithField("codingMethod", config.CodingMethod).Info("Node decoding data")
			droplets := make([][]byte, 0, config.ExpectedChunks)
			if config.CodingMethod == "LT" {
				for _, droplet := range config.ChunksRecByNode {
					if len(droplet) > 0 {
						droplets = append(droplets, droplet)
					}
				}
				_, err = lt.LTDecode(droplets)
			} else if config.CodingMethod == "RS" {
				_, err = rs.RSDecode(config.ChunksRecByNode[:config.ExpectedChunks])
			}

			if (err != nil) && (config.CodingMethod == "LT"){
				log.WithFields(logrus.Fields{"nodeID": nodeID, "Error": err, "length of valid chunks:": len(droplets)}).Error("Node failed to decode data")
				flag = false
				return
			} else if (err != nil) && (config.CodingMethod == "RS") {
				log.WithFields(logrus.Fields{"nodeID": nodeID, "Error": err, "length of valid chunks:": len(config.ChunksRecByNode)}).Error("Node failed to decode data")
				// flag = false
				// return
			}

			log.WithFields(logrus.Fields{"nodeID": nodeID}).Info("Node reconstructed data")

			// outputFilePath := fmt.Sprintf("output/%s_out.txt", config.NodeID)
			// if err := os.WriteFile(outputFilePath, []byte(decodedData), 0644); err != nil {
			// 	log.WithFields(logrus.Fields{"nodeID": nodeID, "Error": err}).Error("Node failed to write reconstructed data to file")
			// 	return
			// }

			for _, peerInfo := range config.ConnectedPeers {
				if peerInfo.ID.String() != nodeID {
					readyKey := fmt.Sprintf("%s-ready", peerInfo.ID.String())
					if _, ok := config.SentChunks.Load(readyKey); !ok {
						SendReady(context.Background(), h, peerInfo, nodeID)
						config.SentChunks.Store(readyKey, struct{}{})
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	} else {
		return
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

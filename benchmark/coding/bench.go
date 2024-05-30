package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/klauspost/reedsolomon"
	goltgoogle "github.com/xm0onh/AVID-go/goltgoogle"
)

// RS parameters
const (
	originalLength = 188762857 // Adjust as needed
	randomSeed     = 42
)

// RS functions
func RSEncode(data string, dataShards, parityShards int) ([][]byte, error) {
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %v", err)
	}
	shards, err := enc.Split([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("failed to split data into shards: %v", err)
	}
	err = enc.Encode(shards)
	if err != nil {
		return nil, fmt.Errorf("failed to encode data shards: %v", err)
	}

	return shards, nil
}

func RSDecode(shards [][]byte, dataShards, parityShards int) (string, error) {
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return "", fmt.Errorf("failed to create decoder: %v", err)
	}

	err = enc.Reconstruct(shards)
	if err != nil {
		return "", fmt.Errorf("failed to reconstruct data shards: %v", err)
	}

	var buf bytes.Buffer
	err = enc.Join(&buf, shards, len(shards[0])*dataShards)
	if err != nil {
		return "", fmt.Errorf("failed to join shards: %v", err)
	}

	decodedData := buf.String()
	trimmedData := decodedData[:len(decodedData)-bytes.Count([]byte(decodedData), []byte{0})]

	return trimmedData, nil
}

// LT functions
func LTEncode(data string, encodedBlockCount, ltSourceBlocks int) ([][]byte, error) {
	c := 1.5
	m := int(c * math.Sqrt(float64(ltSourceBlocks))) // Number of encoded blocks
	delta := 0.01                                    // Failure probability
	degree := goltgoogle.RobustSolitonDistribution(ltSourceBlocks, m, delta)
	random := rand.New(rand.NewSource(randomSeed))
	codec := goltgoogle.NewLubyCodec(ltSourceBlocks, random, degree)

	encodedBlockIDs := make([]int64, encodedBlockCount)
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	encodedBlocks := goltgoogle.EncodeLTBlocks([]byte(data), encodedBlockIDs, codec)

	var chunks [][]byte
	for _, block := range encodedBlocks {
		var buf bytes.Buffer

		// Write the chunk index
		if err := binary.Write(&buf, binary.LittleEndian, int32(block.BlockCode)); err != nil {
			return nil, fmt.Errorf("failed to write chunk index: %v", err)
		}

		// Write the chunk data length
		if err := binary.Write(&buf, binary.LittleEndian, uint32(len(block.Data))); err != nil {
			return nil, fmt.Errorf("failed to write chunk data length: %v", err)
		}

		// Write the chunk data
		if _, err := buf.Write(block.Data); err != nil {
			return nil, fmt.Errorf("failed to write chunk data: %v", err)
		}

		chunks = append(chunks, buf.Bytes())
	}

	return chunks, nil
}

func LTDecode(chunks [][]byte, ltSourceBlocks int) (string, error) {
	c := 1.5
	m := int(c * math.Sqrt(float64(ltSourceBlocks))) // Number of encoded blocks
	delta := 0.01                                    // Failure probability

	degree := goltgoogle.RobustSolitonDistribution(ltSourceBlocks, m, delta)
	random := rand.New(rand.NewSource(randomSeed))
	codec := goltgoogle.NewLubyCodec(ltSourceBlocks, random, degree)

	decoder := codec.NewDecoder(originalLength)

	var ltBlocks []goltgoogle.LTBlock
	for _, chunk := range chunks {
		var blockCode int32
		var dataLength uint32

		buf := bytes.NewBuffer(chunk)

		// Read the chunk index
		if err := binary.Read(buf, binary.LittleEndian, &blockCode); err != nil {
			return "", fmt.Errorf("failed to read chunk index: %v", err)
		}

		// Read the chunk data length
		if err := binary.Read(buf, binary.LittleEndian, &dataLength); err != nil {
			return "", fmt.Errorf("failed to read chunk data length: %v", err)
		}

		// Read the actual chunk data
		data := make([]byte, dataLength)
		if _, err := buf.Read(data); err != nil {
			return "", fmt.Errorf("failed to read chunk data: %v", err)
		}

		ltBlocks = append(ltBlocks, goltgoogle.LTBlock{
			BlockCode: int64(blockCode),
			Data:      data,
		})
	}

	if !decoder.AddBlocks(ltBlocks) {
		return "", fmt.Errorf("insufficient blocks to decode the message")
	}

	decodedData := decoder.Decode()
	if decodedData == nil {
		return "", fmt.Errorf("failed to decode data")
	}

	return string(decodedData), nil
}

// Benchmark functions
func benchmarkRS(data string, dataShards, parityShards int) (int64, int64, float64, error) {
	startTime := time.Now()
	encoded, err := RSEncode(data, dataShards, parityShards)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("RS encoding failed: %v", err)
	}
	encodeTime := time.Since(startTime).Milliseconds()

	startTime = time.Now()
	_, err = RSDecode(encoded, dataShards, parityShards)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("RS decoding failed: %v", err)
	}
	decodeTime := time.Since(startTime).Milliseconds()

	// Calculate RS overhead
	totalRSDataSize := (dataShards + parityShards) * len(encoded[0])
	rsOverhead := float64(totalRSDataSize) / float64(len(data))

	return encodeTime, decodeTime, rsOverhead, nil
}

func benchmarkLT(data string, encodedBlockCount, ltSourceBlocks int) (bool, int64, int64, float64, error) {
	startTime := time.Now()
	encoded, err := LTEncode(data, encodedBlockCount, ltSourceBlocks)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("LT encoding failed: %v", err)
	}
	encodeTime := time.Since(startTime).Milliseconds()

	startTime = time.Now()
	decodeMessage, err := LTDecode(encoded, ltSourceBlocks)
	if err != nil {
		return false, 0, 0, 0, fmt.Errorf("LT decoding failed: %v", err)
	}
	decodeTime := time.Since(startTime).Milliseconds()

	if (decodeMessage) == (data) {
		// Calculate LT overhead
		totalLTDataSize := encodedBlockCount * len(encoded[0])
		ltOverhead := float64(totalLTDataSize) / float64(len(data))

		return true, encodeTime, decodeTime, ltOverhead, nil
	} else {
		return false, 0, 0, 0, fmt.Errorf("LT decoding failed: %v", err)

	}
}

func main() {
	// Load the data file
	data, err := os.ReadFile("./eth_transactions.json")
	if err != nil {
		log.Fatalf("Failed to read data file: %v", err)
	}

	// Benchmark configurations
	rsConfigs := [][2]int{
		{1, 1},
		{5, 2},
		{10, 3},
		{15, 4},
		{20, 5},
	}

	ltConfigs := [][2]int{
		{1, 1},
		{5, 2},
		{10, 3},
		{15, 4},
		{20, 5},
	}

	// RS benchmarks
	fmt.Println("Reed-Solomon (RS) Benchmarks:")
	for _, config := range rsConfigs {
		dataShards, parityShards := config[0], config[1]
		encodeTime, decodeTime, overhead, err := benchmarkRS(string(data), dataShards, parityShards)
		if err != nil {
			log.Printf("RS benchmark failed: %v", err)
			continue
		}
		fmt.Printf("Data Shards: %d, Parity Shards: %d, Encode Time: %.4fs, Decode Time: %.4fs, Overhead: %.2f\n",
			dataShards, parityShards, float64(encodeTime), float64(decodeTime), overhead)
	}

	// LT benchmarks
	fmt.Println("\nLuby Transform (LT) Benchmarks:")
	for _, config := range ltConfigs {
		encodedBlockCount, ltSourceBlocks := config[0], config[1]
		status, encodeTime, decodeTime, overhead, err := benchmarkLT(string(data), encodedBlockCount, ltSourceBlocks)
		if err != nil || !status {
			log.Printf("LT benchmark failed: %v", err)
			continue
		}
		fmt.Printf("Encoded Block Count: %d, LT Source Blocks: %d, Encode Time: %.4fs, Decode Time: %.4fs, Overhead: %.2f\n",
			encodedBlockCount, ltSourceBlocks, float64(encodeTime), float64(decodeTime), overhead)
	}
}

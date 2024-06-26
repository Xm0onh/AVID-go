package luby

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"

	"github.com/xm0onh/AVID-go/config"
	goltgoogle "github.com/xm0onh/AVID-go/goltgoogle"
)

// var degree = goltgoogle.SolitonDistribution(config.LTSourceBlocks)

// var eps = 0.1 // Adjust the epsilon parameter as needed
// var degree = onlineSolitonDistribution(eps)

var n = config.LTSourceBlocks // Number of source blocks
var c = 1.5
var m = int(c * math.Sqrt(float64(n))) // Number of encoded blocks

var delta = 0.01 // Failure probability

var degree = goltgoogle.RobustSolitonDistribution(n, m, delta)

var random = rand.New(rand.NewSource(config.RandomSeed))
var codec = goltgoogle.NewLubyCodec(config.LTSourceBlocks, random, degree)

func LTEncode(data string) ([][]byte, error) {
	fmt.Println(config.LTEncodedBlockCount)
	encodedBlockIDs := make([]int64, config.LTEncodedBlockCount)
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

func LTDecode(chunks [][]byte) (string, error) {
	decoder := codec.NewDecoder(config.OriginalLength)

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

func LTEncodeORG(data string) ([]goltgoogle.LTBlock, error) {
	encodedBlockIDs := make([]int64, config.LTEncodedBlockCount)
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	encodedBlocks := goltgoogle.EncodeLTBlocks([]byte(data), encodedBlockIDs, codec)
	return encodedBlocks, nil
}

func LTDecodeORG(encodedBlocks []goltgoogle.LTBlock) (string, error) {
	decoder := codec.NewDecoder(config.OriginalLength)

	if !decoder.AddBlocks(encodedBlocks) {
		return "", fmt.Errorf("insufficient blocks to decode the message")
	}

	decodedData := decoder.Decode()
	if decodedData == nil {
		return "", fmt.Errorf("failed to decode data")
	}

	return string(decodedData), nil
}

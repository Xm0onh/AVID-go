package luby

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/xm0onh/AVID-go/config"
)

func LTEncode(data string) ([][]byte, error) {
	random := rand.New(rand.NewSource(config.RandomSeed))
	codec := NewLubyCodec(config.LTSourceBlocks, random, SolitonDistribution(config.LTSourceBlocks))

	encodedBlockIDs := make([]int64, config.LTEncodedBlockCount)
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	encodedBlocks := EncodeLTBlocks([]byte(data), encodedBlockIDs, codec)

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
	random := rand.New(rand.NewSource(config.RandomSeed))
	codec := NewLubyCodec(config.LTSourceBlocks, random, SolitonDistribution(config.LTSourceBlocks))
	decoder := codec.NewDecoder(config.OriginalLength)

	var ltBlocks []LTBlock
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

		ltBlocks = append(ltBlocks, LTBlock{
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
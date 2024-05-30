package raptor

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/xm0onh/AVID-go/config"
	goltgoogle "github.com/xm0onh/AVID-go/goltgoogle"
)

// // Configuration variables (adjust these as needed)
// var n = config.RaptorSourceBlocks // Number of source blocks
// var c = 0.1
// var m = int(c * math.Sqrt(float64(n))) // Calculated number of redundant blocks
// var delta = 0.01                       // Failure probability

// var random = rand.New(rand.NewSource(config.RandomSeed))
var codec = goltgoogle.NewRaptorCodec(config.RaptorSourceBlocks, 8) // Adjust alignment size if needed

func RaptorEncode(data string) ([][]byte, error) {
	encodedBlockIDs := make([]int64, config.RaptorEncodedBlockCount)
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	message := []byte(data)
	blocks := goltgoogle.EncodeLTBlocks(message, encodedBlockIDs, codec)

	var chunks [][]byte
	for _, blk := range blocks {
		var buf bytes.Buffer

		// Write the block code
		if err := binary.Write(&buf, binary.LittleEndian, blk.BlockCode); err != nil {
			return nil, fmt.Errorf("failed to write block code: %v", err)
		}

		// Write the chunk data length
		if err := binary.Write(&buf, binary.LittleEndian, uint32(len(blk.Data))); err != nil {
			return nil, fmt.Errorf("failed to write chunk data length: %v", err)
		}

		// Write the chunk data
		if _, err := buf.Write(blk.Data); err != nil {
			return nil, fmt.Errorf("failed to write chunk data: %v", err)
		}

		chunks = append(chunks, buf.Bytes())
	}

	return chunks, nil
}

func RaptorDecode(chunks [][]byte) (string, error) {
	decoder := codec.NewDecoder(config.OriginalLength)

	var ltBlocks []goltgoogle.LTBlock
	for _, chunk := range chunks {
		var blockCode int64
		var dataLength uint32

		buf := bytes.NewBuffer(chunk)

		// Read the block code
		if err := binary.Read(buf, binary.LittleEndian, &blockCode); err != nil {
			return "", fmt.Errorf("failed to read block code: %v", err)
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
			BlockCode: blockCode,
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

package luby

import (
	"fmt"
	"math/rand"

	"github.com/xm0onh/AVID-go/config"
)

// LT Coder
func LTEncode(data string) ([][]byte, error) {
	random := rand.New(rand.NewSource(config.RandomSeed))
	degreeCDF := SolitonDistribution(config.LTSourceBlocks)

	codec := NewLubyCodec(config.LTSourceBlocks, random, degreeCDF)

	encodedBlockIDs := make([]int64, config.LTEncodedBlockCount)
	for i := range encodedBlockIDs {
		encodedBlockIDs[i] = int64(i)
	}

	encodedBlocks := EncodeLTBlocks([]byte(data), encodedBlockIDs, codec)

	chunks := make([][]byte, len(encodedBlocks))
	for i, block := range encodedBlocks {
		chunks[i] = block.Data
	}

	return chunks, nil
}

// LT Decoder
func LTDecode(chunks [][]byte, originalLength int) (string, error) {
	random := rand.New(rand.NewSource(config.RandomSeed))
	degreeCDF := SolitonDistribution(config.LTSourceBlocks)

	codec := NewLubyCodec(config.LTSourceBlocks, random, degreeCDF)

	encodedBlocks := make([]LTBlock, len(chunks))
	for i, chunk := range chunks {
		encodedBlocks[i] = LTBlock{
			BlockCode: int64(i),
			Data:      chunk,
		}
	}

	decoder := codec.NewDecoder(originalLength)

	success := decoder.AddBlocks(encodedBlocks)
	if !success {
		return "", fmt.Errorf("insufficient blocks to decode the message")
	}

	decodedData := decoder.Decode()
	if decodedData == nil {
		return "", fmt.Errorf("failed to decode data")
	}

	return string(decodedData), nil
}

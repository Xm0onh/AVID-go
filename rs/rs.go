
package rs
import (
	"bytes"
	"fmt"

	"github.com/klauspost/reedsolomon"
)
// RS parameters
const dataShards = 3
const parityShards = 2

func RSEncode(data string) ([][]byte, error) {
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

// Decode data using Reed-Solomon decoding
func RSDecode(shards [][]byte) (string, error) {
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

	return buf.String(), nil
}

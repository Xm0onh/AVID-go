package rs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRSEncodeDecode(t *testing.T) {
	originalData := "HelloLibP2P"
	shards, err := RSEncode(originalData)
	assert.NoError(t, err, "RSEncode should not return an error")

	shards[1] = nil

	decodedData, err := RSDecode(shards)
	assert.NoError(t, err, "RSDecode should not return an error")
	assert.Equal(t, originalData, decodedData, "Decoded data should match the original")
}

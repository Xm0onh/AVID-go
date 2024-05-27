package luby

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLTEncodeDecode(t *testing.T) {
	originalData := "Hello, this is a test message for Luby Transform coding!"

	encodedChunks, err := LTEncode(originalData)
	assert.NoError(t, err, "LTEncode should not return an error")

	decodedData, err := LTDecode(encodedChunks, len(originalData))
	assert.NoError(t, err, "LTDecode should not return an error")
	assert.NotNil(t, decodedData, "Expected to decode data")
	assert.Equal(t, originalData, decodedData, "Decoded data should match the original")
}

package luby

import (
	"os"
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/xm0onh/AVID-go/config"
)

func TestLTEncodeDecode(t *testing.T) {
	// Specify the file path
	filePath := "../test.txt"

	// Open the file
	file, err := os.Open(filePath)
	assert.NoError(t, err, "Error opening file")
	defer file.Close()

	// size of file
	fileInfo, err := file.Stat()
	fmt.Println("File size: ", fileInfo.Size())
	originalData, err := os.ReadFile(filePath)
	assert.NoError(t, err, "Error reading file")

	config.OriginalLength = len(originalData)

	// Encode the original data
	chunks, err := LTEncode(string(originalData))
	assert.NoError(t, err, "LTEncode should not return an error")
	assert.NotEmpty(t, chunks, "Encoded chunks should not be empty")
	assert.Equal(t, config.LTEncodedBlockCount, len(chunks), "Number of encoded chunks should match LTEncodedBlockCount")

	// Decode the encoded data
	decodedData, err := LTDecode(chunks)
	assert.NoError(t, err, "LTDecode should not return an error")
	assert.Equal(t, string(originalData), decodedData, "Decoded data should match the original")
}

package raptor

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xm0onh/AVID-go/config"
)

func TestRaptorEncodeDecode(t *testing.T) {
	// Specify the file path
	filePath := "../eth_transactions.json"

	// Open the file
	file, err := os.Open(filePath)
	assert.NoError(t, err, "Error opening file")
	defer file.Close()

	// Get the size of the file
	fileInfo, _ := file.Stat()
	fmt.Println("File size: ", fileInfo.Size())
	originalData, err := os.ReadFile(filePath)
	assert.NoError(t, err, "Error reading file")

	config.OriginalLength = len(originalData)

	// Encode the original data
	timeStart := time.Now()

	encodeTimeStart := time.Now()
	chunks, err := RaptorEncode(string(originalData))
	fmt.Println("leng of chunks: ", len(chunks))
	fmt.Println("Time taken - Encode: ", time.Since(encodeTimeStart))

	assert.NoError(t, err, "RaptorEncode should not return an error")
	assert.NotEmpty(t, chunks, "Encoded chunks should not be empty")
	assert.Equal(t, config.RaptorEncodedBlockCount, len(chunks), "Number of encoded chunks should match RaptorEncodedBlockCount")

	// Decode the encoded data
	decodeTimeStart := time.Now()
	decodedData, err := RaptorDecode(chunks)
	fmt.Println("Time taken - Decode: ", time.Since(decodeTimeStart))

	assert.NoError(t, err, "RaptorDecode should not return an error")
	assert.Equal(t, string(originalData), decodedData, "Decoded data should match the original")
	fmt.Println("Time taken: ", time.Since(timeStart))
}

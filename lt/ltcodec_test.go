package luby

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xm0onh/AVID-go/config"
)

func TestLTEncodeDecode(t *testing.T) {
	// Specify the file path
	filePath := "../eth_transactions.json"

	// Open the file
	file, err := os.Open(filePath)
	assert.NoError(t, err, "Error opening file")
	defer file.Close()

	// size of file
	fileInfo, _ := file.Stat()
	fmt.Println("File size: ", fileInfo.Size())
	originalData, err := os.ReadFile(filePath)
	assert.NoError(t, err, "Error reading file")

	config.OriginalLength = len(originalData)

	// Encode the original data
	timeStart := time.Now()

	encodeTimeStart := time.Now()
	chunks, err := LTEncode(string(originalData))
	fmt.Println("Size of each chunk in byte", len(chunks[0]))
	fmt.Println("Time taken - Encode: ", time.Since(encodeTimeStart))

	assert.NoError(t, err, "LTEncode should not return an error")
	assert.NotEmpty(t, chunks, "Encoded chunks should not be empty")
	// assert.Equal(t, config.LTEncodedBlockCount, len(chunks), "Number of encoded chunks should match LTEncodedBlockCount")

	// Decode the encoded data

	decodeTimeStart := time.Now()
	decodedData, err := LTDecode(chunks)
	fmt.Println("Time taken - Decode: ", time.Since(decodeTimeStart))

	assert.NoError(t, err, "LTDecode should not return an error")
	assert.Equal(t, string(originalData), decodedData, "Decoded data should match the original")
	fmt.Println("Time taken: ", time.Since(timeStart))
}

package rs

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRSEncodeDecode(t *testing.T) {
	filePath := "../test.txt"

	// Open the file
	file, err := os.Open(filePath)
	assert.NoError(t, err, "Error opening file")
	defer file.Close()

	// size of file
	fileInfo, _ := file.Stat()
	fmt.Println("File size: ", fileInfo.Size())

	startTime := time.Now()
	originalData, _ := os.ReadFile(filePath)

	encodeTimeStart := time.Now()
	shards, err := RSEncode(string(originalData))
	fmt.Println("Time taken - Encode: ", time.Since(encodeTimeStart))
	assert.NoError(t, err, "RSEncode should not return an error")
	shards[1] = nil
	// fmt.Println("Shards", shards)
	decodeTimeStart := time.Now()
	decodedData, err := RSDecode(shards)
	fmt.Println("Time taken - Decode: ", time.Since(decodeTimeStart))

	assert.NoError(t, err, "RSDecode should not return an error")
	assert.Equal(t, string(originalData), decodedData, "Decoded data should match the original")
	fmt.Println("Time taken - Total: ", time.Since(startTime))
}

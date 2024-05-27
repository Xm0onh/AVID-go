package luby

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xm0onh/AVID-go/config"
)

func TestLTEncodeDecode(t *testing.T) {
	// Specify the file path
	filePath := "../test.txt"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Get the file information
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}

	// Get the size of the file
	fileSize := fileInfo.Size()
	fmt.Printf("The size of the file is: %d bytes\n", fileSize)
	originalData := "Hello, this is a test message for Luby Transform coding!"
	// var RandomSeed = 42 // Ensure the same seed is used for consistency

	// Encode the original data
	chunks, err := LTEncode(originalData)
	assert.NoError(t, err, "LTEncode should not return an error")
	assert.NotEmpty(t, chunks, "Encoded chunks should not be empty")
	fmt.Println("Encoded chunks", chunks)
	// Decode the encoded data
	config.OriginalLength = len(originalData)
	decodedData, err := LTDecode(chunks)
	fmt.Println("Decoded data", decodedData)
	assert.NoError(t, err, "LTDecode should not return an error")
	assert.Equal(t, originalData, decodedData, "Decoded data should match the original")
}

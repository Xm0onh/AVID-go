package main

func DivideIntoChunks(data string, numChunks int) []string {
	chunkLength := len(data) / numChunks
	chunks := make([]string, numChunks)

	for i := 0; i < numChunks; i++ {
		start := i * chunkLength
		end := start + chunkLength
		if i == numChunks-1 {
			end = len(data) // Make sure the last chunk includes any remaining characters
		}
		chunks[i] = data[start:end]
	}
	return chunks
}

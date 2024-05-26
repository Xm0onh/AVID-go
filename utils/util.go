package util



func DivideIntoChunks(data string, numChunks int) []string {
	chunkLength := len(data) / numChunks
	chunks := make([]string, numChunks)

	for i := 0; i < numChunks; i++ {
		start := i * chunkLength
		end := start + chunkLength
		if i == numChunks-1 {
			end = len(data) 
		}
		chunks[i] = data[start:end]
	}
	return chunks
}

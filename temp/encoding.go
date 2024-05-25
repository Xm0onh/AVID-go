package main

// import (
// 	"bytes"
// 	"crypto/sha256"

// 	"github.com/klauspost/reedsolomon"
// )

// // EncodeRS encodes the file using Reed-Solomon codes
// func EncodeRS(data []byte, n, k int) ([][]byte, [][]byte, error) {
// 	enc, err := reedsolomon.New(k, n-k)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	shards, err := enc.Split(data)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	err = enc.Encode(shards)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	hashes := make([][]byte, len(shards))
// 	for i, shard := range shards {
// 		hash := sha256.Sum256(shard)
// 		hashes[i] = hash[:]
// 	}

// 	return shards, hashes, nil
// }

// // DecodeRS decodes the shards back into the original file
// func DecodeRS(shards [][]byte, k int) ([]byte, error) {
// 	enc, err := reedsolomon.New(k, len(shards)-k)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ok, err := enc.Verify(shards)
// 	if err != nil || !ok {
// 		err = enc.Reconstruct(shards)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	buf := new(bytes.Buffer)
// 	err = enc.Join(buf, shards, len(shards[0])*k)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return buf.Bytes(), nil
// }

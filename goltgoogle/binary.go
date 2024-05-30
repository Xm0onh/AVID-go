// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fountain

import (
	"math/rand"
)

// Random binary fountain code. In this code, the constituent source Blocks in
// a code Block are selected randomly and independently.

// BinaryCodec contains the codec information for the random binary fountain
// encoder and decoder.
// Implements fountain.Codec
type binaryCodec struct {
	// numSourceBlocks is the number of source Blocks (N) the source message is split into.
	numSourceBlocks int
}

// NewBinaryCodec returns a codec implementing the binary fountain code,
// where source Blocks composing each LT Block are chosen randomly and independently.
func NewBinaryCodec(numSourceBlocks int) Codec {
	return &binaryCodec{numSourceBlocks: numSourceBlocks}
}

// SourceBlocks returns the number of source Blocks used in the codec.
func (c *binaryCodec) SourceBlocks() int {
	return c.numSourceBlocks
}

// PickIndices finds the source indices for a code Block given an ID and
// a random seed. Uses the Mersenne Twister internally.
func (c *binaryCodec) PickIndices(codeBlockIndex int64) []int {
	random := rand.New(NewMersenneTwister(codeBlockIndex))

	var indices []int
	for b := 0; b < c.SourceBlocks(); b++ {
		if random.Intn(2) == 1 {
			indices = append(indices, b)
		}
	}

	return indices
}

// GenerateIntermediateBlocks simply returns the partition of the input message
// into source Blocks. It does not perform any additional precoding.
func (c *binaryCodec) GenerateIntermediateBlocks(message []byte, numBlocks int) []Block {
	long, short := PartitionBytes(message, c.numSourceBlocks)
	source := EqualizeBlockLengths(long, short)

	return source
}

// NewDecoder creates a new binary fountain code decoder
func (c *binaryCodec) NewDecoder(messageLength int) Decoder {
	return newBinaryDecoder(c, messageLength)
}

// binaryDecoder is the state required to decode a combinatoric fountain
// code message.
type binaryDecoder struct {
	codec         binaryCodec
	messageLength int

	// The sparse equation matrix used for decoding.
	matrix sparseMatrix
}

// newBinaryDecoder creates a new decoder for a particular message.
// The codec parameters used to create the original encoding Blocks must be provided.
// The decoder is only valid for decoding code Blocks for a particular message.
func newBinaryDecoder(c *binaryCodec, length int) *binaryDecoder {
	return &binaryDecoder{
		codec:         *c,
		messageLength: length,
		matrix: sparseMatrix{
			coeff: make([][]int, c.numSourceBlocks),
			v:     make([]Block, c.numSourceBlocks),
		}}
}

// AddBlocks adds a set of encoded Blocks to the decoder. Returns true if the
// message can be fully decoded. False if there is insufficient information.
func (d *binaryDecoder) AddBlocks(Blocks []LTBlock) bool {
	for i := range Blocks {
		d.matrix.addEquation(d.codec.PickIndices(Blocks[i].BlockCode),
			Block{Data: Blocks[i].Data})
	}
	return d.matrix.determined()
}

// Decode extracts the decoded message from the decoder. If the decoder does
// not have sufficient information to produce an output, returns a nil slice.
func (d *binaryDecoder) Decode() []byte {
	if !d.matrix.determined() {
		return nil
	}

	d.matrix.reduce()

	lenLong, lenShort, numLong, numShort := partition(d.messageLength, d.codec.numSourceBlocks)
	return d.matrix.reconstruct(d.messageLength, lenLong, lenShort, numLong, numShort)
}

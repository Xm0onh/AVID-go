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
	"math"
	"math/rand"
)

// Implemention of Online Codes. See
// http://cs.nyu.edu/web/Research/TechReports/TR2002-833/TR2002-833.pdf
// After Maymounkov and Mazieres
//
// The input message is a sequence of bytes which is split into N Blocks.
//
// The constraints on the codec parameters are that 0.55*e*q*N >= q, probably
// like q*2, which implies that 0.55*e*N >= 1 or e*N >= 1.82.
//
// Taking epsilon small (i.e. 0.01), this means N > 200 or so. A more
// reasonable approach gives N > 400 or so for e=0.01.
//
// What this means for small texts is that e should be fairly big, like say
// 0.3 or something. That means 0.3*N > 4 or so, meaning N is like 12-15.
// This still allows us to get good probability of recovery. It means source
// symbols are small, but (e/2)^(q+1) can still be 10^-12 with e=0.3 if
// q is large enough (like 15).
//
// The number of code Blocks expected to provide almost certain message
// recovery is (1+epsilon)(1+0.55*Q*epsilon)N where N is the number of source Blocks.
//
// For large texts, the primary concern is picking N such that the packet
// size is convenient. Ideally epsilon is quite small, meaning not much extra
// Data transmission required, and since there are many Blocks, that can be
// done without increasing q very much.

// onlineCodec contains the parameters for application of an online code, typically
// for a particular known message.
// Recommended parameters for large NumSourceBlocks: Epsilon=0.01, Quality=3.
// Note that it requires quite large numbers (that is, thousands) of input Blocks
// to approach optimality. For example, NumSourceBlocks=1000 requires about 3%
// overhead at these settings to achieve recovery error rate of 1e-8.
// Implements fountain.Codec
type onlineCodec struct {
	// epsilon is the suboptimality parameter. ("Efficiency" or "e")
	// A message of N Blocks can be decoded with high probability
	// from (1+3*epsilon)*numSourceBlocks received Blocks.
	epsilon float64

	// quality is the decoder quality factor ("q"). This parameter influences the
	// failure rate of the decoder.
	// Given (1+3*epsilon)*N Blocks, the algorithm will fail with probability
	// (epsilon/2)^(quality+1)
	quality int

	// numSourceBlocks is the number of source Blocks ("N") to construct from the
	// input message. This parameter interacts with the message length to set the
	// packet size, so should be picked with that in mind.
	numSourceBlocks int

	// randomSeed is a source of randomness for selecting auxiliary encoding Blocks.
	// This seeds a psuedorandom source identically for both encoding and decoding.
	randomSeed int64

	// cdf is the cumulative distribution function of the degree distribution.
	cdf []float64
}

// NewOnlineCodec creates a new encoder for an Online code.
// epsilon is the suboptimality parameter. ("Efficiency" or "e")
// A message of N Blocks can be decoded with high probability
// from (1+3*epsilon)*numSourceBlocks received Blocks.
// quality is the decoder quality factor ("q"). This parameter influences the
// failure rate of the decoder.
// Given (1+3*epsilon)*N Blocks, the algorithm will fail with probability
// (epsilon/2)^(quality+1)
// seed is the random seed used to pick auxiliary encoding Blocks.
func NewOnlineCodec(sourceBlocks int, epsilon float64, quality int, seed int64) Codec {
	return &onlineCodec{
		epsilon:         epsilon,
		quality:         quality,
		numSourceBlocks: sourceBlocks,
		randomSeed:      seed,
		cdf:             onlineSolitonDistribution(epsilon)}
}

// SourceBlocks returns the number of source Blocks into which the codec will
// partition an input message.
func (c *onlineCodec) SourceBlocks() int {
	return c.numSourceBlocks
}

// numAuxBlocks returns the number of auxiliary Blocks to create for the outer
// encoding.
func (c onlineCodec) numAuxBlocks() int {
	// Note: equation is from the paper.
	return int(math.Ceil(0.55 * float64(c.quality) * c.epsilon * float64(c.numSourceBlocks)))
}

// estimateDecodeBlocksNeeded returns a rough lower bound on the number of decode
// Blocks likely needed to successfully decode a message. This number is about
// (1+epsilon)(NumSourceBlocks + numAuxBlocks)
func (c onlineCodec) estimateDecodeBlocksNeeded() int {
	return int(math.Ceil((1 + c.epsilon) * float64(c.numSourceBlocks+c.numAuxBlocks())))
}

// GenerateIntermediateBlocks finds a set of auxiliary encoding Blocks using an
// LT process, which it then appends to the original set of message Blocks.
func (c *onlineCodec) GenerateIntermediateBlocks(message []byte, numBlocks int) []Block {
	src, aux := generateOuterEncoding(message, *c)
	intermediate := make([]Block, len(src), len(src)+len(aux))
	copy(intermediate, src)
	intermediate = append(intermediate, aux...)
	return intermediate
}

// generateOuterEncoding creates the source and auxiliary Blocks after section
// 3.1 of http://pdos.csail.mit.edu/~petar/papers/maymounkov-bigdown-lncs.ps
// Returns two slices of Blocks: the original source Blocks and the auxiliary
// Blocks.
// Basic idea: the auxiliary Blocks are randomly composed of the source Blocks
// and then used to generate code Blocks. This makes recovery of the full
// original message from code Blocks more robust.
func generateOuterEncoding(message []byte, codec onlineCodec) ([]Block, []Block) {
	numAuxBlocks := codec.numAuxBlocks()
	long, short := PartitionBytes(message, codec.numSourceBlocks)
	source := EqualizeBlockLengths(long, short)

	aux := make([]Block, numAuxBlocks)
	// Ensure all aux Blocks have the same length as the source Blocks,
	// even if they don't happen to get loaded with Data.
	for i := range aux {
		aux[i].Padding = source[0].length()
	}

	random := rand.New(NewMersenneTwister(codec.randomSeed))
	for i := 0; i < codec.numSourceBlocks; i++ {
		touchAuxBlocks := sampleUniform(random, codec.quality, numAuxBlocks)
		for _, j := range touchAuxBlocks {
			aux[j].xor(source[i])
		}
	}

	return source, aux
}

// generateCodeBlock creates a new code symbol, which is the XOR of
// outer Blocks [b_k1, b_k2, b_k3, ... b_kd]
// Where the sequence k1, k2, k3, ..., kd is provided in the indices.
func generateCodeBlock(source []Block, aux []Block, indices []int) Block {
	var symbol Block

	for _, i := range indices {
		if i < len(source) {
			symbol.xor(source[i])
		} else {
			symbol.xor(aux[i-len(source)])
		}
	}

	return symbol
}

// PickIndices finds the source indices for a code Block given an ID using
// the CDF for the online degree distribution.
func (c *onlineCodec) PickIndices(codeBlockIndex int64) []int {
	random := rand.New(NewMersenneTwister(codeBlockIndex))

	degree := pickDegree(random, c.cdf)
	// Pick Blocks from the augmented set of original+aux Blocks produced
	// by GenerateIntermediateBlocks.
	s := sampleUniform(random, degree, c.SourceBlocks()+c.numAuxBlocks())
	return s
}

// encodeOnlineBlocks creates a set of online code Blocks given the ids provided.
// An easy way to generate the ids is to pick a pseudo-random sequence and then
// just grab the first M members of the sequence.
// The characteristic of an online code is that this method may be called
// repeatedly with different ids, generating different code Blocks for the same
// message, all of which can then be used interchangeably by the decoder.
// For each code Block, we pick a random set of outer-encoding Blocks to XOR
// to compose it.
func encodeOnlineBlocks(message []byte, ids []int64, codec onlineCodec) []LTBlock {
	source, aux := generateOuterEncoding(message, codec)
	Blocks := make([]LTBlock, len(ids))
	for i := range Blocks {
		indices := codec.PickIndices(ids[i])
		Block := generateCodeBlock(source, aux, indices)
		Blocks[i].BlockCode = ids[i]
		Blocks[i].Data = make([]byte, source[0].length())
		copy(Blocks[i].Data, Block.Data)
	}
	return Blocks
}

// onlineDecoder is the state required for decoding a particular message prepared
// with the onlineCodec. It must be initialized with the same parameters
// used for encoding, as well as the expected message length.
// Implements fountain.Decoder
type onlineDecoder struct {
	codec         *onlineCodec
	messageLength int

	// The sparse equation matrix used for decoding.
	matrix sparseMatrix
}

// NewDecoder creates an online transform decoder
func (c *onlineCodec) NewDecoder(messageLength int) Decoder {
	return newOnlineDecoder(c, messageLength)
}

// newOnlineDecoder creates a new decoder for a particular message. The codec
// parameters as well as the original message length must be provided. The
// decoder is only valid for decoding Blocks for a particular source message.
func newOnlineDecoder(c *onlineCodec, length int) *onlineDecoder {
	d := &onlineDecoder{codec: c, messageLength: length}

	numAuxBlocks := c.numAuxBlocks()
	d.matrix.coeff = make([][]int, c.numSourceBlocks+numAuxBlocks)
	d.matrix.v = make([]Block, c.numSourceBlocks+numAuxBlocks)

	// Now we add the initial auxiliary equations into the decode matrix.
	// These come in as synthetic decode Blocks, which have value 0 and
	// coefficient bits set indicating their constituent outer Blocks.
	auxBlockComposition := make([][]int, numAuxBlocks)
	random := rand.New(NewMersenneTwister(c.randomSeed))
	for i := 0; i < c.numSourceBlocks; i++ {
		touchAuxBlocks := sampleUniform(random, c.quality, numAuxBlocks)
		for _, j := range touchAuxBlocks {
			auxBlockComposition[j] = append(auxBlockComposition[j], i)
		}
	}
	for i := range auxBlockComposition {
		auxBlockComposition[i] = append(auxBlockComposition[i], i+c.numSourceBlocks)
	}
	// Note: these composition slices are guaranteed sorted since we added constituent
	// source Blocks in order, followed by the aux Block index. So we can now just
	// add them to the equation matrix as if they were received.

	for i := range auxBlockComposition {
		d.matrix.addEquation(auxBlockComposition[i], Block{})
	}

	return d
}

// AddBlocks adds a set of encoded Blocks to the decoder. Returns true if the
// message can be fully decoded. False if there is insufficient information.
func (d *onlineDecoder) AddBlocks(Blocks []LTBlock) bool {
	for i := range Blocks {
		indices := d.codec.PickIndices(Blocks[i].BlockCode)
		d.matrix.addEquation(indices, Block{Data: Blocks[i].Data})
	}
	return d.matrix.determined()
}

// Decode extracts the decoded message from the decoder. If the decoder does
// not have sufficient information to produce an output, returns a nil slice.
func (d *onlineDecoder) Decode() []byte {
	if !d.matrix.determined() {
		return nil
	}

	d.matrix.reduce()

	lenLong, lenShort, numLong, numShort := partition(d.messageLength, d.codec.numSourceBlocks)
	return d.matrix.reconstruct(d.messageLength, lenLong, lenShort, numLong, numShort)
}

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

// A Block represents a contiguous range of Data being encoded or decoded,
// or a Block of coded Data. Details of how the source text is split into Blocks
// is governed by the particular fountain code used.
type Block struct {
	// Data content of this source or code Block.
	Data []byte

	// How many Padding bytes this Block has at the end.
	Padding int
}

// newBlock creates a new Block with a given length. The Block will initially be
// all Padding.
func newBlock(len int) *Block {
	return &Block{Padding: len}
}

// length returns the length of the Block in bytes. Counts Data bytes as well
// as any Padding.
func (b *Block) length() int {
	return len(b.Data) + b.Padding
}

func (b *Block) empty() bool {
	return b.length() == 0
}

// A common operation is to XOR entire code Blocks together with other Blocks.
// When this is done, Padding bytes count as 0 (that is XOR identity), and the
// destination Block will be modified so that its Data is large enough to
// contain the result of the XOR.
func (b *Block) xor(a Block) {
	if len(b.Data) < len(a.Data) {
		var inc = len(a.Data) - len(b.Data)
		b.Data = append(b.Data, make([]byte, inc)...)
		if b.Padding > inc {
			b.Padding -= inc
		} else {
			b.Padding = 0
		}
	}

	for i := 0; i < len(a.Data); i++ {
		b.Data[i] ^= a.Data[i]
	}
}

// PartitionBytes partitions an input text into a sequence of p Blocks. The
// sizes of the Blocks will be given by the partition() function. The last
// Block may have Padding.
// Return values: the slice of longer Blocks, the slice of shorter Blocks.
// Within each Block slice, all will have uniform lengths.
func PartitionBytes(in []byte, p int) ([]Block, []Block) {
	sliceIntoBlocks := func(in []byte, num, length int) ([]Block, []byte) {
		Blocks := make([]Block, num)
		for i := range Blocks {
			if len(in) > length {
				Blocks[i].Data, in = in[:length], in[length:]
			} else {
				Blocks[i].Data, in = in, []byte{}
			}
			if len(Blocks[i].Data) < length {
				Blocks[i].Padding = length - len(Blocks[i].Data)
			}
		}
		return Blocks, in
	}

	lenLong, lenShort, numLong, numShort := partition(len(in), p)
	long, in := sliceIntoBlocks(in, numLong, lenLong)
	short, _ := sliceIntoBlocks(in, numShort, lenShort)
	return long, short
}

// EqualizeBlockLengths adds Padding to all short Blocks to make them equal in
// size to the long Blocks. The caller should ensure that all the longBlocks
// have the same length.
// Returns a Block slice containing all the long and short Blocks.
func EqualizeBlockLengths(longBlocks, shortBlocks []Block) []Block {
	if len(longBlocks) == 0 {
		return shortBlocks
	}
	if len(shortBlocks) == 0 {
		return longBlocks
	}

	for i := range shortBlocks {
		shortBlocks[i].Padding += longBlocks[0].length() - shortBlocks[i].length()
	}

	Blocks := make([]Block, len(longBlocks)+len(shortBlocks))
	copy(Blocks, longBlocks)
	copy(Blocks[len(longBlocks):], shortBlocks)
	return Blocks
}

// sparseMatrix is the Block decoding Data structure. It is a sparse matrix of
// XOR equations. The coefficients are the indices of the source Blocks which
// are XORed together to produce the values. So if equation _i_ of the matrix is
// Block_0 ^ Block_2 ^ Block_3 ^ Block_9 = [0xD2, 0x38]
// that would be represented as coeff[i] = [0, 2, 3, 9], v[i].Data = [0xD2, 0x38]
// Example: The sparse coefficient matrix
// | 0 1 1 0 |
// | 0 1 0 0 |
// | 0 0 1 1 |
// | 0 0 0 1 |
// would be represented as
// [ [ 1, 2],
//
//	[ 1 ],
//	[ 2, 3],
//	[ 3 ]]
//
// Every row has M[i][0] >= i. If we added components [2] to this matrix, it
// would replace the M[2] row ([2, 3]), and then the resulting component vector
// after cancellation against that row ([3]) would be used instead of the
// original equation. In this case, M[3] is already populated, so the new
// equation is redundant (and could be used for ECC, theoretically), but if we
// didn't have an entry in M[3], it would be placed there.
// The values were omitted from this discussion, but they follow along by doing
// XOR operations as the components are reduced during insertion.
type sparseMatrix struct {
	coeff [][]int
	v     []Block
}

// xorRow performs a reduction of the given candidate equation (indices, b)
// with the specified matrix row (index s). It does so by XORing the values,
// and then taking the symmetric difference of the coefficients of that matrix
// row and the provided indices. (That is, the "set XOR".) Assumes both
// coefficient slices are sorted.
func (m *sparseMatrix) xorRow(s int, indices []int, b Block) ([]int, Block) {
	b.xor(m.v[s])

	var newIndices []int
	coeffs := m.coeff[s]
	var i, j int
	for i < len(coeffs) && j < len(indices) {
		index := indices[j]
		if coeffs[i] == index {
			i++
			j++
		} else if coeffs[i] < index {
			newIndices = append(newIndices, coeffs[i])
			i++
		} else {
			newIndices = append(newIndices, index)
			j++
		}
	}

	newIndices = append(newIndices, coeffs[i:]...)
	newIndices = append(newIndices, indices[j:]...)
	return newIndices, b
}

// addEquation adds an XOR equation to the decode matrix. The online decode
// strategy is a variant of that of Bioglio, Grangetto, and Gaeta
// (http://www.di.unito.it/~bioglio/Papers/CL2009-lt.pdf) It maintains the
// invariant that either coeff[i][0] == i or len(coeff[i]) == 0. That is, while
// adding an equation to the matrix, it ensures that the decode matrix remains
// triangular.
func (m *sparseMatrix) addEquation(components []int, b Block) {
	// This loop reduces the incoming equation by XOR until it either fits into
	// an empty row in the decode matrix or is discarded as redundant.
	for len(components) > 0 && len(m.coeff[components[0]]) > 0 {
		s := components[0]
		if len(components) >= len(m.coeff[s]) {
			components, b = m.xorRow(s, components, b)
		} else {
			// Swap the existing row for the new one, reduce the existing one and
			// see if it fits elsewhere.
			components, m.coeff[s] = m.coeff[s], components
			b, m.v[s] = m.v[s], b
		}
	}

	if len(components) > 0 {
		m.coeff[components[0]] = components
		m.v[components[0]] = b
	}
}

// Check to see if the decode matrix is fully specified. This is true when
// all rows have non-empty coefficient slices.
// TODO(gbillock): is there a weakness here if an auxiliary Block is unpopulated?
func (m *sparseMatrix) determined() bool {
	for _, r := range m.coeff {
		if len(r) == 0 {
			return false
		}
	}
	return true
}

// reduce performs Gaussian Elimination over the whole matrix. Presumes
// the matrix is triangular, and that the method is not called unless there is
// enough Data for a solution.
// TODO(gbillock): Could profitably do this online as well?
func (m *sparseMatrix) reduce() {
	for i := len(m.coeff) - 1; i >= 0; i-- {
		for j := 0; j < i; j++ {
			ci, cj := m.coeff[i], m.coeff[j]
			for k := 1; k < len(cj); k++ {
				if cj[k] == ci[0] {
					m.v[j].xor(m.v[i])
					continue
				}
			}
		}
		// All but the leading coefficient in the rows have been reduced out.
		m.coeff[i] = m.coeff[i][0:1]
	}
}

// reconstruct pastes the fully reduced values in the sparse matrix result column
// into a new byte array and returns it. The length/number parameters are typically
// those given by partition().
// lenLong is how many long Blocks there are.
// lenShort is how many short Blocks there are (following the long Blocks).
// numLong is how many bytes are in the long Blocks.
// numShort is how many bytes the short Blocks are.
func (m *sparseMatrix) reconstruct(totalLength, lenLong, lenShort, numLong, numShort int) []byte {
	out := make([]byte, totalLength)
	out = out[0:0]
	for i := 0; i < numLong; i++ {
		out = append(out, m.v[i].Data[0:lenLong]...)
	}
	for i := numLong; i < numLong+numShort; i++ {
		out = append(out, m.v[i].Data[0:lenShort]...)
	}

	return out
}

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
	"reflect"
	"testing"
)

func printEncoding(source []Block, aux []Block, t *testing.T) {
	t.Log("Outer Encoding Blocks")
	t.Log("---------------------")
	kb := 0
	for s := range source {
		t.Log("src", kb, source[s].Data)
		kb++
	}
	for a := range aux {
		t.Log("aux", kb, aux[a].Data)
		kb++
	}
}

func TestOnlineBlocks(t *testing.T) {
	c := onlineCodec{epsilon: 0.01, quality: 5, numSourceBlocks: 6, randomSeed: 200}
	if c.numAuxBlocks() != 1 {
		t.Errorf("Got %d aux Blocks, want 1", c.numAuxBlocks())
	}
	message := []byte("abcdefghijklmnopqrstuvwxyz")

	source, aux := generateOuterEncoding(message, c)
	printEncoding(source, aux, t)

	if source[0].Data[0] != 'a' {
		t.Errorf("Source Data should start with message beginning. Got %s", source[0].Data)
	}

	Block := generateCodeBlock(source, aux, []int{4})
	if !reflect.DeepEqual(Block.Data, source[4].Data) {
		t.Errorf("Single Data Block is %v, should be %v", Block.Data, source[4].Data)
	}
	Block = generateCodeBlock(source, aux, []int{2, 5, 6})
	if Block.Data[0] != 107^119^7 {
		t.Errorf("XOR Data Block got %v, should be %v", Block.Data[0], 107^119^7)
	}
	t.Log("Block =", Block)

	codec := NewOnlineCodec(6, 0.01, 5, 200)
	ltBlocks := EncodeLTBlocks(message, []int64{252}, codec)
	indices := codec.PickIndices(252)
	if !reflect.DeepEqual(indices, []int{4}) {
		t.Errorf("Indices for 252 are %v, should be [4]", indices)
	}
	if !reflect.DeepEqual(ltBlocks[0].Data, source[4].Data) {
		t.Errorf("Single Data Block is %v, should be %v", ltBlocks[0].Data, source[4].Data)
	}
	t.Log("Block =", ltBlocks[0])
}

func TestDecoder(t *testing.T) {
	c := NewOnlineCodec(13, 0.3, 10, 200).(*onlineCodec)
	message := []byte("abcdefghijklmnopqrstuvwxyz")
	ids := make([]int64, 45)
	random := rand.New(rand.NewSource(8923489))
	for i := range ids {
		ids[i] = int64(random.Intn(100000))
	}
	source, aux := generateOuterEncoding(message, *c)
	printEncoding(source, aux, t)

	Blocks := encodeOnlineBlocks(message, ids, *c)
	t.Log("Blocks =", Blocks)

	d := newOnlineDecoder(c, len(message))

	for i := 0; i < 16; i++ {
		d.AddBlocks([]LTBlock{Blocks[i]})
		if testing.Verbose() {
			printMatrix(d.matrix, t)
		}
	}

	d.matrix.reduce()
	t.Log("REDUCE")
	printMatrix(d.matrix, t)

	decoded := d.Decode()
	printMatrix(d.matrix, t)
	if !reflect.DeepEqual(decoded, message) {
		t.Errorf("Got %v, want %v", decoded, message)
	}
}

func TestDecoderBlockTable(t *testing.T) {
	c := NewOnlineCodec(13, 0.3, 10, 0).(*onlineCodec)
	if c.numAuxBlocks() != 22 {
		t.Errorf("Got %d aux Blocks, want 22", c.numAuxBlocks())
	}
	needed := c.estimateDecodeBlocksNeeded()
	if needed != 46 {
		t.Errorf("Got %d Blocks expected to be needed, want 17", needed)
	}

	message := []byte("abcdefghijklmnopqrstuvwxyz")
	random := rand.New(rand.NewSource(8234923))

	moreBlocksNeeded := 0
	for i := 0; i < 100; i++ {
		c.randomSeed = random.Int63()
		r := rand.New(rand.NewSource(random.Int63()))
		ids := make([]int64, 45)
		for i := range ids {
			ids[i] = int64(r.Intn(100000))
		}
		Blocks := encodeOnlineBlocks(message, ids, *c)

		d := newOnlineDecoder(c, len(message))
		d.AddBlocks(Blocks[0:30])
		if !d.matrix.determined() {
			moreBlocksNeeded++
			d.AddBlocks(Blocks[31:46])
		}
		decoded := d.Decode()
		if !reflect.DeepEqual(decoded, message) {
			t.Errorf("Got %v, want %v", decoded, message)
		}
	}

	if moreBlocksNeeded > 2 {
		t.Errorf("Needed too many high-Block-count decoding sequences: %d", moreBlocksNeeded)
	}
}

func TestDecodeMessageTable(t *testing.T) {
	c := NewOnlineCodec(10, 0.2, 7, 0).(*onlineCodec)
	random := rand.New(rand.NewSource(8234982))
	for i := 0; i < 100; i++ {
		c.randomSeed = random.Int63()
		r := rand.New(rand.NewSource(random.Int63()))
		messageLen := r.Intn(1000) + 1000
		message := make([]byte, messageLen)
		for j := 0; j < len(message); j++ {
			message[j] = byte(r.Intn(200))
		}
		ids := make([]int64, 50)
		for i := range ids {
			ids[i] = int64(r.Intn(100000))
		}
		Blocks := encodeOnlineBlocks(message, ids, *c)

		d := newOnlineDecoder(c, len(message))
		d.AddBlocks(Blocks[0:25])
		if !d.matrix.determined() {
			t.Errorf("Message should be determined after 25 Blocks")
		} else {
			decoded := d.Decode()
			if !reflect.DeepEqual(decoded, message) {
				t.Errorf("Incorrect message decode. Length=%d", len(message))
			}
		}
	}
}

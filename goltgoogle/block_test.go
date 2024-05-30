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
	"bytes"
	"reflect"
	"testing"
)

func TestBlockLength(t *testing.T) {
	var lengthTests = []struct {
		b   Block
		len int
	}{
		{Block{}, 0},
		{Block{[]byte{1, 0, 1}, 0}, 3},
		{Block{[]byte{1, 0, 1}, 1}, 4},
	}

	for _, i := range lengthTests {
		if i.b.length() != i.len {
			t.Errorf("Length of b is %d, should be %d", i.b.length(), i.len)
		}
		if (i.len == 0) != i.b.empty() {
			t.Errorf("Emptiness check error. Got %v, want %v", i.b.empty(), i.len == 0)
		}
	}
}

func TestBlockXor(t *testing.T) {
	var xorTests = []struct {
		a   Block
		b   Block
		out Block
	}{
		{Block{[]byte{1, 0, 1}, 0}, Block{[]byte{1, 1, 1}, 0}, Block{[]byte{0, 1, 0}, 0}},
		{Block{[]byte{1}, 0}, Block{[]byte{0, 14, 6}, 0}, Block{[]byte{1, 14, 6}, 0}},
		{Block{}, Block{[]byte{100, 200}, 0}, Block{[]byte{100, 200}, 0}},
		{Block{[]byte{}, 5}, Block{[]byte{0, 1, 0}, 0}, Block{[]byte{0, 1, 0}, 2}},
		{Block{[]byte{}, 5}, Block{[]byte{0, 1, 0, 2, 3}, 0}, Block{[]byte{0, 1, 0, 2, 3}, 0}},
		{Block{[]byte{}, 5}, Block{[]byte{0, 1, 0, 2, 3, 7}, 0}, Block{[]byte{0, 1, 0, 2, 3, 7}, 0}},
		{Block{[]byte{1}, 4}, Block{[]byte{0, 1, 0, 2, 3, 7}, 0}, Block{[]byte{1, 1, 0, 2, 3, 7}, 0}},
	}

	for _, i := range xorTests {
		t.Logf("...Testing %v XOR %v", i.a, i.b)
		originalLength := i.a.length()
		i.a.xor(i.b)
		if i.a.length() < originalLength {
			t.Errorf("Length shrunk. Got %d, want length >= %d", i.a.length(), originalLength)
		}
		if len(i.a.Data) != len(i.b.Data) {
			t.Errorf("a and b Data should be same length after xor. a len=%d, b len=%d", len(i.a.Data), len(i.b.Data))
		}

		if !bytes.Equal(i.a.Data, i.out.Data) {
			t.Errorf("XOR value is %v : should be %v", i.a.Data, i.out.Data)
		}
	}
}

func TestPartitionBytes(t *testing.T) {
	a := make([]byte, 100)
	for i := 0; i < len(a); i++ {
		a[i] = byte(i)
	}

	var partitionTests = []struct {
		numPartitions     int
		lenLong, lenShort int
	}{
		{11, 1, 10},
		{3, 1, 2},
	}

	for _, i := range partitionTests {
		t.Logf("Partitioning %v into %d", a, i.numPartitions)
		long, short := PartitionBytes(a, i.numPartitions)
		if len(long) != i.lenLong {
			t.Errorf("Got %d long Blocks, should have %d", len(long), i.lenLong)
		}
		if len(short) != i.lenShort {
			t.Errorf("Got %d short Blocks, should have %d", len(short), i.lenShort)
		}
		if short[len(short)-1].Padding != 0 {
			t.Errorf("Should fit Blocks exactly, have last Padding %d", short[len(short)-1].Padding)
		}
		if long[0].Data[0] != 0 {
			t.Errorf("Long Block should be first. First value is %v", long[0].Data)
		}
	}
}

func TestEqualizeBlockLengths(t *testing.T) {
	b := []byte("abcdefghijklmnopq")
	var equalizeTests = []struct {
		numPartitions int
		length        int
		Padding       int
	}{
		{1, 17, 0},
		{2, 9, 1},
		{3, 6, 1},
		{4, 5, 1},
		{5, 4, 1},
		{6, 3, 1},
		{7, 3, 1},
		{8, 3, 1},
		{9, 2, 1},
		{10, 2, 1},
		{16, 2, 1},
		{17, 1, 0},
	}

	for _, i := range equalizeTests {
		long, short := PartitionBytes(b, i.numPartitions)
		Blocks := EqualizeBlockLengths(long, short)
		if len(Blocks) != i.numPartitions {
			t.Errorf("Got %d Blocks, should have %d", len(Blocks), i.numPartitions)
		}
		for k := range Blocks {
			if Blocks[k].length() != i.length {
				t.Errorf("Got Block length %d for Block %d, should be %d",
					Blocks[0].length(), k, i.length)
			}
		}
		if Blocks[len(Blocks)-1].Padding != i.Padding {
			t.Errorf("Padding of last Block is %d, should be %d",
				Blocks[len(Blocks)-1].Padding, i.Padding)
		}
	}
}

func printMatrix(m sparseMatrix, t *testing.T) {
	t.Log("------- matrix -----------")
	for i := range m.coeff {
		t.Logf("%v = %v\n", m.coeff[i], m.v[i].Data)
	}
}

func TestMatrixXorRow(t *testing.T) {
	var xorRowTests = []struct {
		arow   []int
		r      []int
		result []int
	}{
		{[]int{0, 1}, []int{2, 3}, []int{0, 1, 2, 3}},
		{[]int{0, 1}, []int{1, 2, 3}, []int{0, 2, 3}},
		{[]int{}, []int{1, 2, 3}, []int{1, 2, 3}},
		{[]int{1, 2, 3}, []int{}, []int{1, 2, 3}},
		{[]int{1}, []int{2}, []int{1, 2}},
		{[]int{1}, []int{1}, []int{}},
		{[]int{1, 2}, []int{1, 2, 3, 4}, []int{3, 4}},
		{[]int{3, 4}, []int{1, 2, 3, 4}, []int{1, 2}},
		{[]int{1, 2, 3, 4}, []int{1, 2}, []int{3, 4}},
		{[]int{0, 1, 2, 3, 4}, []int{1, 2}, []int{0, 3, 4}},
		{[]int{3, 4}, []int{1, 2, 3, 4, 5}, []int{1, 2, 5}},
		{[]int{3, 4, 8}, []int{1, 2, 3, 4, 5}, []int{1, 2, 5, 8}},
	}

	for _, test := range xorRowTests {
		m := sparseMatrix{coeff: [][]int{test.arow}, v: []Block{Block{[]byte{1}, 0}}}

		testb := Block{[]byte{2}, 0}
		test.r, testb = m.xorRow(0, test.r, testb)

		// Needed since under DeepEqual the nil and the empty slice are not equal.
		if test.r == nil {
			test.r = make([]int, 0)
		}
		if !reflect.DeepEqual(test.r, test.result) {
			t.Errorf("XOR row result got %v, should be %v", test.r, test.result)
		}
		if !reflect.DeepEqual(testb, Block{[]byte{3}, 0}) {
			t.Errorf("XOR row Block got %v, should be %v", testb, Block{[]byte{3}, 0})
		}
	}
}

func TestMatrixBasic(t *testing.T) {
	m := sparseMatrix{coeff: [][]int{{}, {}}, v: []Block{Block{}, Block{}}}

	m.addEquation([]int{0}, Block{Data: []byte{1}})
	if m.determined() {
		t.Errorf("2-row matrix should not be determined after 1 equation")
		printMatrix(m, t)
	}

	m.addEquation([]int{0, 1}, Block{Data: []byte{2}})
	if !m.determined() {
		t.Errorf("2-row matrix should be determined after 2 equations")
		printMatrix(m, t)
	}

	printMatrix(m, t)

	if !reflect.DeepEqual(m.coeff[0], []int{0}) ||
		!reflect.DeepEqual(m.v[0].Data, []byte{1}) {
		t.Errorf("Equation 0 got (%v = %v), want ([0] = [1])", m.coeff[0], m.v[0].Data)
	}
	if !reflect.DeepEqual(m.coeff[1], []int{1}) ||
		!reflect.DeepEqual(m.v[1].Data, []byte{3}) {
		t.Errorf("Equation 1 got (%v = %v), want ([1] = [3])", m.coeff[0], m.v[0].Data)
	}

	m.reduce()
	if !reflect.DeepEqual(m.coeff[0], []int{0}) ||
		!reflect.DeepEqual(m.v[0].Data, []byte{1}) {
		t.Errorf("Equation 0 got (%v = %v), want ([0] = [1])", m.coeff[0], m.v[0].Data)
	}
	if !reflect.DeepEqual(m.coeff[1], []int{1}) ||
		!reflect.DeepEqual(m.v[1].Data, []byte{3}) {
		t.Errorf("Equation 1 got (%v = %v), want ([1] = [3])", m.coeff[0], m.v[0].Data)
	}
}

func TestMatrixLarge(t *testing.T) {
	m := sparseMatrix{coeff: make([][]int, 4), v: make([]Block, 4)}

	m.addEquation([]int{2, 3}, Block{Data: []byte{1}})
	m.addEquation([]int{2}, Block{Data: []byte{2}})
	if m.determined() {
		t.Errorf("4-row matrix should not be determined after 2 equations")
		printMatrix(m, t)
	}
	printMatrix(m, t)

	// Should have triangular entries in {2} and {3} now.
	if len(m.coeff[2]) != 1 || m.v[2].Data[0] != 2 {
		t.Errorf("Equation 2 got %v = %v, should be [2] = [2]", m.coeff[2], m.v[2])
	}
	if len(m.coeff[3]) != 1 || m.v[3].Data[0] != 3 {
		t.Errorf("Equation 3 got %v = %v, should be [3] = [3]", m.coeff[3], m.v[3])
	}
	if len(m.coeff[0]) != 0 || len(m.coeff[1]) != 0 {
		t.Errorf("Equations 0 and 1 should be empty")
		printMatrix(m, t)
	}

	m.addEquation([]int{0, 1, 2, 3}, Block{Data: []byte{4}})
	if m.determined() {
		t.Errorf("4-row matrix should not be determined after 3 equations")
		printMatrix(m, t)
	}

	m.addEquation([]int{3}, Block{Data: []byte{3}})
	if m.determined() {
		t.Errorf("4-row matrix should not be determined after redundant equation")
		printMatrix(m, t)
	}

	m.addEquation([]int{0, 2}, Block{Data: []byte{8}})
	if !m.determined() {
		t.Errorf("4-row matrix should be determined after 4 equations")
		printMatrix(m, t)
	}

	// The matrix should now have entries in rows 0 and 1, but not equal to the
	// original equations.
	printMatrix(m, t)
	if !reflect.DeepEqual(m.coeff[0], []int{0, 2}) {
		t.Errorf("Got %v for coeff[0], expect [0, 2]", m.coeff[0])
	}
	if !reflect.DeepEqual(m.coeff[1], []int{1, 3}) {
		t.Errorf("Got %v for coeff[1], expect [1, 3]", m.coeff[1])
	}
}

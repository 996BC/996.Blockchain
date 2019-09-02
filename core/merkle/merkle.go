package merkle

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/996BC/996.Blockchain/utils"
)

type MerkleLeafs [][]byte

func (l MerkleLeafs) Len() int {
	return len(l)
}

func (l MerkleLeafs) Less(i, j int) bool {
	return bytes.Compare(l[i], l[j]) < 0
}

func (l MerkleLeafs) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// SortAndComputeRoot sort the input and ComputeRoot
func SortAndComputeRoot(leafs MerkleLeafs) ([]byte, error) {
	SortByteArray(leafs)
	return ComputeRoot(leafs)
}

// ComputeRoot calcultes the merkle root of byte array
// The input parameter will be modified and be careful to reuse it
func ComputeRoot(leafs MerkleLeafs) ([]byte, error) {
	if leafs.Len() == 0 {
		return nil, fmt.Errorf("nil input")
	}

	if len(leafs) == 1 {
		// If the node only has one child, its hash is equal to its child's hash.
		return leafs[0], nil
	}

	harray := leafs
	for true {
		arrayLen := len(harray)
		if arrayLen == 1 {
			break
		}

		overwriteIndex := 0
		for i := 0; i < arrayLen; {
			hashOfTwoNode := harray[i]

			if i+1 < arrayLen {
				hashOfTwoNode = utils.Hash(append(harray[i], harray[i+1]...))
			}
			harray[overwriteIndex] = hashOfTwoNode

			i += 2
			overwriteIndex++
		}

		harray = harray[:overwriteIndex]
	}

	return harray[0], nil
}

// SortByteArray sorts byte array in increasing order
func SortByteArray(leafs MerkleLeafs) {
	if leafs.Len() == 0 {
		return
	}

	sort.Sort(leafs)
}

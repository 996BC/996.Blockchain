package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

func TestInsert(t *testing.T) {
	cases := []struct {
		origin       []int
		insertValue  int
		expectResult []int
	}{
		{[]int{}, 1, []int{1}}, //0
		{[]int{2}, 1, []int{1, 2}},
		{[]int{2}, 2, []int{2, 2}},
		{[]int{2}, 3, []int{2, 3}},
		{[]int{2, 4}, 1, []int{1, 2, 4}},
		{[]int{2, 4}, 2, []int{2, 2, 4}}, //5
		{[]int{2, 4}, 3, []int{2, 3, 4}},
		{[]int{2, 4}, 4, []int{2, 4, 4}},
		{[]int{2, 4}, 5, []int{2, 4, 5}},
		{[]int{2, 4, 6}, 1, []int{1, 2, 4, 6}},
		{[]int{2, 4, 6}, 2, []int{2, 2, 4, 6}}, //10
		{[]int{2, 4, 6}, 3, []int{2, 3, 4, 6}},
		{[]int{2, 4, 6}, 4, []int{2, 4, 4, 6}},
		{[]int{2, 4, 6}, 5, []int{2, 4, 5, 6}},
		{[]int{2, 4, 6}, 6, []int{2, 4, 6, 6}},
		{[]int{2, 4, 6}, 7, []int{2, 4, 6, 7}}, //15
		{[]int{2, 4, 6, 8}, 1, []int{1, 2, 4, 6, 8}},
		{[]int{2, 4, 6, 8}, 2, []int{2, 2, 4, 6, 8}},
		{[]int{2, 4, 6, 8}, 3, []int{2, 3, 4, 6, 8}},
		{[]int{2, 4, 6, 8}, 4, []int{2, 4, 4, 6, 8}},
		{[]int{2, 4, 6, 8}, 5, []int{2, 4, 5, 6, 8}},
		{[]int{2, 4, 6, 8}, 6, []int{2, 4, 6, 6, 8}}, //20
		{[]int{2, 4, 6, 8}, 7, []int{2, 4, 6, 7, 8}},
		{[]int{2, 4, 6, 8}, 8, []int{2, 4, 6, 8, 8}},
		{[]int{2, 4, 6, 8}, 9, []int{2, 4, 6, 8, 9}},
	}

	for i, cs := range cases {
		evp := &evidencePool{}
		for _, oriNum := range cs.origin {
			evp.evds = append(evp.evds, &weightedEvidence{nil, big.NewInt(int64(oriNum))})
		}
		insertEvd := &weightedEvidence{nil, big.NewInt(int64(cs.insertValue))}
		evp.insert(insertEvd)

		var result []int
		for _, evd := range evp.evds {
			result = append(result, int(evd.weight.Int64()))
		}

		for j := 0; j < len(result); j++ {
			if err := utils.TCheckInt(fmt.Sprintf("[%d] index %d number", i, j),
				cs.expectResult[j], result[j]); err != nil {
				t.Fatal(err)
			}
		}
	}
}

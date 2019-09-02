package merkle

import (
	"fmt"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

func init() {
	utils.SetLogLevel(utils.LogDebugLevel)
}

func TestComputeMerkleRoot(t *testing.T) {
	var cases = []struct {
		in     []string // hash
		expect string   // hex of MerkleRoot
	}{
		{[]string{}, ""},
		{[]string{""}, ""},
		{[]string{"hash_111"}, "686173685F313131"},
		{[]string{"hash_111", "hash_222"}, "608CE07EDED7DBC19061F61A113A478B20F8454FA6E7BB76E28FCE508B8657A6"},
		{[]string{"hash_111", "hash_222", "hash_333"}, "7ECB95FAA39AF2F13D303ADA13415D7B7C12AB2B90880DD190DCA026FC7E3359"},
		{[]string{"hash_111", "hash_222", "hash_333", "hash_444"}, "2D0C475AB3C2B7D6EEC94CEF5A5990AFCA22CF3583D69048275F8382485D1DBA"},
		{[]string{"hash_111", "hash_222", "hash_333", "hash_444", "hash_555"},
			"5B94B011897FFA8766429F17B7978D5178E55A1E168A24C1034532C70A87101C"},
	}

	for i, c := range cases {
		input := toByteArray(c.in)

		output, _ := ComputeRoot(input)
		if err := utils.TCheckString(fmt.Sprintf("[%d] root", i), c.expect, utils.ToHex(output)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSortByteArray(t *testing.T) {
	var cases = []struct {
		in     []string
		expect []string
	}{
		{nil, nil},
		{[]string{}, []string{}},
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "z", "b"}, []string{"a", "b", "z"}},
	}

	for i, c := range cases {
		ba := toByteArray(c.in)
		SortByteArray(ba)

		if err := utils.TCheckInt(fmt.Sprintf("[%d] array length", i), len(c.expect), len(ba)); err != nil {
			t.Fatal(err)
		}

		for j := 0; j < len(ba); j++ {
			if err := utils.TCheckString(fmt.Sprintf("[%d] index %d character", i, j),
				c.expect[j], string(ba[j])); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func toByteArray(sa []string) [][]byte {
	var result [][]byte
	for _, s := range sa {
		result = append(result, []byte(s))
	}
	return result
}

package blockchain

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

func TestTargetToDiff(t *testing.T) {
	var cases = []struct {
		target         uint32
		length         uint32
		prefixDiff     uint32
		expectedBitLen int
	}{
		{0x00000000, 0x00, 0x000000, calExpectedBitLen(0x00, 0x000000)},
		{0xFFFFFFFF, 0xFF, 0xFFFFFF, calExpectedBitLen(0xFF, 0xFFFFFF)},
		{0x01000001, 0x01, 0x000001, calExpectedBitLen(0x01, 0x000001)},
		{0x1F000001, 0x1F, 0x000001, calExpectedBitLen(0x1F, 0x000001)},
		{0x01100001, 0x01, 0x100001, calExpectedBitLen(0x01, 0x100001)},
		{0x1F100001, 0x1F, 0x100001, calExpectedBitLen(0x1F, 0x100001)},
		{0xE9100000, 0xE9, 0x100000, calExpectedBitLen(0xE9, 0x100000)},
	}

	for i, c := range cases {
		diff := TargetToDiff(c.target)

		if err := utils.TCheckInt(fmt.Sprintf("[%d] difficulty bit length", i),
			c.expectedBitLen, diff.BitLen()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiffToTarget(t *testing.T) {
	var cases = []struct {
		diffBits       string
		expectedTarget uint32
	}{
		{"0", genTargetFromBits("0")},
		{"0001", genTargetFromBits("0001")},
		{"00000000000000000000000000000000000", genTargetFromBits("00000000000000000000000000000000000")},
		{"00000000000000000000000000000000000001", genTargetFromBits("00000000000000000000000000000000000001")},
		{"10000000000000000000000000000000000000", genTargetFromBits("10000000000000000000000000000000000000")},
		{"11010001111011101110000111100101011111011101110001", genTargetFromBits("11010001111011101110000111100101011111011101110001")},
		{"11111111111111111111111111111111111111111111111111", genTargetFromBits("11111111111111111111111111111111111111111111111111")},
	}

	for i, c := range cases {
		diff := genDiffFromBits(c.diffBits)
		target := DiffToTarget(diff)

		if err := utils.TCheckUint32(fmt.Sprintf("[%d] target", i),
			c.expectedTarget, target); err != nil {
			t.Fatal(err)
		}
	}
}

func calExpectedBitLen(length uint32, prefixDiff uint32) int {
	prefixBitLen := big.NewInt(int64(prefixDiff)).BitLen()
	if length < diffPrefixBitsLen {
		return prefixBitLen
	}
	return prefixBitLen + int(length-diffPrefixBitsLen)
}

func genDiffFromBits(bits string) *big.Int {
	result := big.NewInt(0)
	i := strings.Index(bits, "1")
	if i == -1 {
		return big.NewInt(0)
	}

	for i < len(bits) {
		result.Lsh(result, 1)
		if bits[i] == '1' {
			result.Add(result, big.NewInt(1))
		}
		i++
	}
	return result
}

func genTargetFromBits(bits string) uint32 {
	var result uint32

	bitsToNum := func(a string) uint32 {
		// a length must small than 24
		var ret uint32
		for _, c := range a {
			ret = ret << 1
			if c == '1' {
				ret++
			}
		}
		return ret
	}

	begin1 := strings.Index(bits, "1")
	if begin1 == -1 {
		return 0
	}

	length := len(bits) - begin1
	var prefix uint32
	if length < diffPrefixBitsLen {
		prefix = bitsToNum(bits[begin1:])
	} else {
		prefix = bitsToNum(bits[begin1 : begin1+diffPrefixBitsLen])
	}

	result = uint32(length) << diffPrefixBitsLen
	result = result | prefix
	return result
}

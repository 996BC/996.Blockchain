package storage

import (
	"bytes"
	"testing"

	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

func TestBlockHeader(t *testing.T) {
	headerParams := cp.NewBlockHeaderParams()
	header := cp.GenBlockHeaderFromParams(headerParams)
	height := uint64(100)

	blockHeader := NewBlockHeader(header, height)
	blockHeaderBytes := blockHeader.Marshal()

	rBlockHeader, err := UnmarshalBlockHeader(bytes.NewReader(blockHeaderBytes))
	if err != nil {
		t.Fatalf("unmarshal block header failed:%v\n", err)
	}

	if err := cp.CheckBlockHeader(rBlockHeader.BlockHeader, headerParams); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint64("height", height, rBlockHeader.Height); err != nil {
		t.Fatal(err)
	}
}

func TestBlock(t *testing.T) {
	evdsHash := [][]byte{
		utils.Hash([]byte("1111")),
		utils.Hash([]byte("22222")),
		utils.Hash([]byte("333333")),
	}

	block := NewBlock(evdsHash)
	blockBytes := block.Marshal()

	rBlock, err := UnmarshalBlock(bytes.NewReader(blockBytes))
	if err != nil {
		t.Fatalf("unmarshal block failed:%v\n", err)
	}

	if err := utils.TCheckInt("evidence size", len(evdsHash), len(rBlock.EvdsHash)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(evdsHash); i++ {
		if err := utils.TCheckBytes("evidence hash", evdsHash[i], rBlock.EvdsHash[i]); err != nil {
			t.Fatal(err)
		}
	}
}

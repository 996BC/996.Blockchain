package db

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

var dbTestVar = &struct {
	dbPath string

	genesis         *cp.Block
	secondBlock     *cp.Block
	thirdEmptyBlock *cp.Block
	genesisHeight   uint64
	secondHeight    uint64
	thirdHeight     uint64

	evidenceA       *cp.Evidence
	evidenceB       *cp.Evidence
	evidenceAHeight uint64
	evidenceBHeight uint64

	// its miner is the 2th block miner
	// and one of its evidence(evidenceC) owner is the evidenceA owner
	fourthBlock                 *cp.Block
	evidenceC                   *cp.Evidence
	fourthHeight                uint64
	secondBlockMinerExpectScore uint64
}{
	genesisHeight:               1,
	secondHeight:                2,
	thirdHeight:                 3,
	fourthHeight:                4,
	secondBlockMinerExpectScore: 2,
}

func init() {
	tv := dbTestVar

	tv.genesis = cp.GenBlockFromParams(cp.NewBlockParams(false))
	tv.secondBlock = cp.GenBlockFromParams(cp.NewBlockParams(false))
	tv.thirdEmptyBlock = cp.GenBlockFromParams(cp.NewBlockParams(true))

	tv.evidenceA = tv.genesis.Evds[0]
	tv.evidenceAHeight = tv.genesisHeight
	tv.evidenceB = tv.secondBlock.Evds[0]
	tv.evidenceBHeight = tv.secondHeight

	tv.fourthBlock = cp.GenBlockFromParams(cp.NewBlockParams(false))
	tv.evidenceC = tv.fourthBlock.Evds[0]
	copy(tv.fourthBlock.Miner, tv.secondBlock.Miner)
	copy(tv.evidenceC.PubKey, tv.evidenceA.PubKey)
}

func setup() {
	tv := dbTestVar

	runningDir, err := os.Getwd()
	if err != nil {
		logger.Fatalln(err)
	}

	tv.dbPath = runningDir + "/db_test_tmp"
	if err := os.MkdirAll(tv.dbPath, 0700); err != nil {
		logger.Fatal("create tmp directory failed:%v\n", err)
	}

	if err := Init(tv.dbPath); err != nil {
		logger.Fatal("initialize db failed:%v\n", err)
	}
}

func cleanup() {
	tv := dbTestVar

	Close()
	if err := os.RemoveAll(tv.dbPath); err != nil {
		logger.Fatal("remove tmp directory failed:%v\n", err)
	}
}

func insertGenesis(t *testing.T) {
	tv := dbTestVar

	if err := PutGenesis(tv.genesis); err != nil {
		t.Fatalf("insert genesis failed:%v\n", err)
	}
}

func insertTestData(t *testing.T) {
	tv := dbTestVar

	insertGenesis(t)
	if err := PutBlock(tv.secondBlock, tv.secondHeight); err != nil {
		t.Fatalf("insert the second block failed:%v\n", err)
	}
	if err := PutBlock(tv.thirdEmptyBlock, tv.thirdHeight); err != nil {
		t.Fatalf("insert the third block failed:%v\n", err)
	}
	if err := PutBlock(tv.fourthBlock, tv.fourthHeight); err != nil {
		t.Fatalf("insert the fourth block failed:%v\n", err)
	}
}

func TestGenesis(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()

	// exists test
	if HasGenesis() {
		t.Fatalf("expect no genesis block exists\n")
	}

	insertGenesis(t)
	if !HasGenesis() {
		t.Fatalf("expect genesis block exists\n")
	}

	// recover test
	dbBlock, hash, err := GetBlockViaHeight(tv.genesisHeight)
	if err != nil {
		t.Fatal(err)
	}

	h := tv.genesis.GetSerializedHash()
	if err := utils.TCheckBytes("block hash", h, hash); err != nil {
		t.Fatal(err)
	}

	checkBlock(t, "", tv.genesis, dbBlock)
}

func TestGetHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		height     uint64
		expectHash []byte
	}{
		{tv.genesisHeight, tv.genesis.GetSerializedHash()},
		{tv.secondHeight, tv.secondBlock.GetSerializedHash()},
		{tv.thirdHeight, tv.thirdEmptyBlock.GetSerializedHash()},
	}

	for i, cs := range cases {
		result, err := GetHash(cs.height)
		if err != nil {
			t.Fatal(err)
		}

		if err := utils.TCheckBytes(fmt.Sprintf("[%d] test case", i),
			cs.expectHash, result); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetHeaderViaHeight(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		height       uint64
		expectHeader *cp.BlockHeader
		expectHash   []byte
	}{
		{tv.genesisHeight, tv.genesis.BlockHeader, tv.genesis.GetSerializedHash()},
		{tv.secondHeight, tv.secondBlock.BlockHeader, tv.secondBlock.GetSerializedHash()},
		{tv.thirdHeight, tv.thirdEmptyBlock.BlockHeader, tv.thirdEmptyBlock.GetSerializedHash()},
	}

	for i, cs := range cases {
		header, hash, err := GetHeaderViaHeight(cs.height)
		if err != nil {
			t.Fatal(err)
		}

		checkHeader(t, fmt.Sprintf("[%d] ", i), cs.expectHeader, header)

		if err := utils.TCheckBytes(fmt.Sprintf("[%d] hash", i), cs.expectHash, hash); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetHeaderViaHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		hash         []byte
		expectHeader *cp.BlockHeader
		expectHeight uint64
	}{
		{tv.genesis.GetSerializedHash(), tv.genesis.BlockHeader, tv.genesisHeight},
		{tv.secondBlock.GetSerializedHash(), tv.secondBlock.BlockHeader, tv.secondHeight},
		{tv.thirdEmptyBlock.GetSerializedHash(), tv.thirdEmptyBlock.BlockHeader, tv.thirdHeight},
	}

	for i, cs := range cases {
		header, height, err := GetHeaderViaHash(cs.hash)
		if err != nil {
			t.Fatal(err)
		}

		checkHeader(t, fmt.Sprintf("[%d] ", i), cs.expectHeader, header)

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] height ", i), cs.expectHeight, height); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetBlockViaHeight(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		height      uint64
		expectBlock *cp.Block
		expectHash  []byte
	}{
		{tv.genesisHeight, tv.genesis, tv.genesis.GetSerializedHash()},
		{tv.secondHeight, tv.secondBlock, tv.secondBlock.GetSerializedHash()},
		{tv.thirdHeight, tv.thirdEmptyBlock, tv.thirdEmptyBlock.GetSerializedHash()},
	}

	for i, cs := range cases {
		block, hash, err := GetBlockViaHeight(cs.height)
		if err != nil {
			t.Fatal(err)
		}

		checkBlock(t, fmt.Sprintf("[%d] block ", i), cs.expectBlock, block)

		if err := utils.TCheckBytes(fmt.Sprintf("[%d] hash", i), cs.expectHash, hash); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetBlockViaHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		hash         []byte
		expectBlock  *cp.Block
		expectHeight uint64
	}{
		{tv.genesis.GetSerializedHash(), tv.genesis, tv.genesisHeight},
		{tv.secondBlock.GetSerializedHash(), tv.secondBlock, tv.secondHeight},
		{tv.thirdEmptyBlock.GetSerializedHash(), tv.thirdEmptyBlock, tv.thirdHeight},
	}

	for i, cs := range cases {
		block, height, err := GetBlockViaHash(cs.hash)
		if err != nil {
			t.Fatal(err)
		}

		checkBlock(t, fmt.Sprintf("[%d] block ", i), cs.expectBlock, block)

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] height ", i), cs.expectHeight, height); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetEvidenceViaHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		hash           []byte
		expectEvidence *cp.Evidence
		expectHeight   uint64
	}{
		{tv.evidenceA.Hash, tv.evidenceA, tv.evidenceAHeight},
		{tv.evidenceB.Hash, tv.evidenceB, tv.evidenceBHeight},
	}

	for i, cs := range cases {
		evidence, height, err := GetEvidenceViaHash(cs.hash)
		if err != nil {
			t.Fatal(err)
		}

		checkEvidence(t, fmt.Sprintf("[%d] evidence ", i), cs.expectEvidence, evidence)

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] height ", i), cs.expectHeight, height); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetEvidenceViaKey(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		key           []byte
		expectedEvds  [][]byte
		expectHeights []uint64
	}{
		{tv.evidenceA.PubKey, [][]byte{tv.evidenceA.Hash, tv.evidenceC.Hash}, []uint64{tv.evidenceAHeight}},
		{tv.evidenceB.PubKey, [][]byte{tv.evidenceB.Hash}, []uint64{tv.evidenceBHeight}},
	}

	for i, cs := range cases {
		evds, heights, err := GetEvidenceViaKey(cs.key)
		if err != nil {
			t.Fatal(err)
		}

		if err := utils.TCheckInt(fmt.Sprintf("[%d] evidence size", i), len(cs.expectedEvds), len(evds)); err != nil {
			t.Fatal(err)
		}

		for j := len(evds); j < len(evds); j++ {
			if err := utils.TCheckBytes(fmt.Sprintf("[%d-%d] evidence hash ", i, j), cs.expectedEvds[j], evds[j]); err != nil {
				t.Fatal(err)
			}
			if err := utils.TCheckUint64(fmt.Sprintf("[%d-%d] height", i, j), cs.expectHeights[j], heights[j]); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestHasEvidence(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	if HasEvidence([]byte("not_exist_evidence")) {
		t.Fatal("expect not found")
	}

	if !HasEvidence(tv.evidenceA.Hash) {
		t.Fatal("expect evidenceA exist")
	}
}

func TestGetScoreViaKey(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		miner       []byte
		expectScore uint64
	}{
		{tv.thirdEmptyBlock.Miner, 1},
		{tv.secondBlock.Miner, tv.secondBlockMinerExpectScore},
		{[]byte("zero_score_miner"), 0},
	}

	for i, cs := range cases {
		score, err := GetScoreViaKey(cs.miner)
		if err != nil {
			t.Fatal(err)
		}

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] score", i), cs.expectScore, score); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetLatestHeight(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()

	_, err := GetLatestHeight()
	if err == nil {
		t.Fatalf("expect error\n")
	}

	insertGenesis(t)
	height, _ := GetLatestHeight()
	if err := utils.TCheckUint64("height", tv.genesisHeight, height); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.secondBlock, tv.secondHeight)
	height, _ = GetLatestHeight()
	if err := utils.TCheckUint64("height", tv.secondHeight, height); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.thirdEmptyBlock, tv.thirdHeight)
	height, _ = GetLatestHeight()
	if err := utils.TCheckUint64("height", tv.thirdHeight, height); err != nil {
		t.Fatal(err)
	}
}

func TestGetLatestHeader(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()

	_, _, _, err := GetLatestHeader()
	if err == nil {
		t.Fatalf("expect error\n")
	}

	insertGenesis(t)
	header, height, hash, _ := GetLatestHeader()
	checkHeader(t, "latest header", tv.genesis.BlockHeader, header)
	if err := utils.TCheckUint64("1st latest header height", tv.genesisHeight, height); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("1st latest header hash", tv.genesis.GetSerializedHash(), hash); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.secondBlock, tv.secondHeight)
	header, height, hash, _ = GetLatestHeader()
	checkHeader(t, "latest header", tv.secondBlock.BlockHeader, header)
	if err := utils.TCheckUint64("2th latest header height", tv.secondHeight, height); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("2th latest header hash", tv.secondBlock.GetSerializedHash(), hash); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.thirdEmptyBlock, tv.thirdHeight)
	header, height, hash, _ = GetLatestHeader()
	checkHeader(t, "latest header", tv.thirdEmptyBlock.BlockHeader, header)
	if err := utils.TCheckUint64("3th latest header height", tv.thirdHeight, height); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("3th latest header hash", tv.thirdEmptyBlock.GetSerializedHash(), hash); err != nil {
		t.Fatal(err)
	}
}

func checkEvidence(t *testing.T, prefix string, expect *cp.Evidence, result *cp.Evidence) {
	expectBytes := expect.Marshal()
	resultBytes := result.Marshal()

	if !bytes.Equal(expectBytes, resultBytes) {
		t.Fatalf("%s evidence mismatch, expect %v, result %v\n", prefix, expect, result)
	}
}

func checkHeader(t *testing.T, prefix string, expect *cp.BlockHeader, result *cp.BlockHeader) {
	expectBytes := expect.Marshal()
	resultBytes := result.Marshal()

	if !bytes.Equal(expectBytes, resultBytes) {
		t.Fatalf("%s header mismatch, expect %v, result %v\n", prefix, expect, result)
	}
}

func checkBlock(t *testing.T, prefix string, expect *cp.Block, result *cp.Block) {
	expectBytes := expect.Marshal()
	resultBytes := result.Marshal()

	if !bytes.Equal(expectBytes, resultBytes) {
		t.Fatalf("%s block mismatch, expect %v, result %v\n", prefix, expect, result)
	}
}

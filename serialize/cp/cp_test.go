package cp

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

func TestHead(t *testing.T) {
	head := NewHeadV1(MsgSyncReq)
	headBytes := head.Marshal()

	rHead, err := UnmarshalHead(bytes.NewReader(headBytes))
	if err != nil {
		t.Fatalf("unmarshal Head failed:%v\n", err)
	}

	if err := utils.TCheckUint8("version", CoreProtocolV1, rHead.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("type", MsgSyncReq, rHead.Type); err != nil {
		t.Fatal(err)
	}
}

func TestEvidence(t *testing.T) {
	ep := NewEvidenceParams()

	evidence := GenEvidenceFromParams(ep)
	evidenceBytes := evidence.Marshal()

	rEvidence, err := UnmarshalEvidence(bytes.NewReader(evidenceBytes))
	if err != nil {
		t.Fatalf("unmarshal evidence failed:%v\n", err)
	}

	if err := CheckEvidence(rEvidence, ep); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceNextNonce(t *testing.T) {
	ep := NewEvidenceParams()
	evidence := GenEvidenceFromParams(ep)
	nonce := uint32(0xFFF0)

	var result []*big.Int
	var expect []*big.Int
	caseSize := 128

	for i := 0; i < caseSize; i++ {
		evidence.SetNonce(nonce + uint32(i))
		expect = append(expect, evidence.GetPow())
	}
	expectFinalNonce := evidence.Nonce

	// recover nonce
	evidence.SetNonce(nonce)
	for i := 0; i < caseSize; i++ {
		// copy the result
		result = append(result, big.NewInt(0).Set(evidence.NextNonce()))
	}
	resultFinalNonce := evidence.Nonce

	for i := 0; i < caseSize; i++ {
		if err := utils.TCheckBigInt("pow value", expect[i], result[i]); err != nil {
			t.Fatalf("case %d failed:%v\n", i, err)
		}
	}

	if err := utils.TCheckUint32("final nonce", expectFinalNonce, resultFinalNonce); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceVerify(t *testing.T) {
	failedEvd := GenEvidenceFromParams(NewEvidenceParams())
	var evd *Evidence

	evd = GenEvidenceFromParams(NewEvidenceParams())
	if err := evd.Verify(); err != nil {
		t.Fatalf("expect valid, but %v\n", err)
	}

	evd = GenEvidenceFromParams(NewEvidenceParams())
	evd.Version = 0
	if err := evd.Verify(); err == nil {
		t.Fatal("expect version error")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	evd = GenEvidenceFromParams(NewEvidenceParams())
	evd.PubKey = failedEvd.PubKey
	if err := evd.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	evd = GenEvidenceFromParams(NewEvidenceParams())
	evd.Sig = failedEvd.Sig
	if err := evd.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	evd = GenEvidenceFromParams(NewEvidenceParams())
	evd.Description = failedEvd.Description
	if err := evd.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	evd = GenEvidenceFromParams(NewEvidenceParams())
	evd.Hash = failedEvd.Hash
	if err := evd.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}
}

func TestBlockHeader(t *testing.T) {
	bp := NewBlockHeaderParams()

	blockHeader := GenBlockHeaderFromParams(bp)
	blockHeaderBytes := blockHeader.Marshal()

	rBlockHeader, err := UnmarshalBlockHeader(bytes.NewReader(blockHeaderBytes))
	if err != nil {
		t.Fatalf("unmarshal block header failed:%v\n", err)
	}

	if err := utils.TCheckInt64("time", blockHeader.Time, rBlockHeader.Time); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlockHeader(rBlockHeader, bp); err != nil {
		t.Fatal(err)
	}

	if err := rBlockHeader.Verify(); err != nil {
		t.Fatalf("expect valid:%v\n", err)
	}
}

func TestBlockHeaderNextNonce(t *testing.T) {
	bp := NewBlockHeaderParams()
	blockHeader := GenBlockHeaderFromParams(bp)
	nonce := uint32(0xFFF0)

	var result []*big.Int
	var expect []*big.Int
	caseSize := 128

	for i := 0; i < caseSize; i++ {
		blockHeader.SetNonce(nonce + uint32(i))
		expect = append(expect, blockHeader.GetPow())
	}
	expectFinalNonce := blockHeader.Nonce

	// recover nonce
	blockHeader.SetNonce(nonce)
	for i := 0; i < caseSize; i++ {
		// copy the result
		result = append(result, big.NewInt(0).Set(blockHeader.NextNonce()))
	}
	resultFinalNonce := blockHeader.Nonce

	for i := 0; i < caseSize; i++ {
		if err := utils.TCheckBigInt("pow value", expect[i], result[i]); err != nil {
			t.Fatalf("case %d failed:%v\n", i, err)
		}
	}

	if err := utils.TCheckUint32("final nonce", expectFinalNonce, resultFinalNonce); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyBlock(t *testing.T) {
	bp := NewBlockParams(true)
	block := GenBlockFromParams(bp)
	blockBytes := block.Marshal()
	t.Logf("empty block size:%d\n", len(blockBytes))

	rBlock, err := UnmarshalBlock(bytes.NewReader(blockBytes))
	if err != nil {
		t.Fatalf("unmarshal block failed:%v", err)
	}

	if err := utils.TCheckInt64("time", block.Time, rBlock.Time); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlock(rBlock, bp); err != nil {
		t.Fatal(err)
	}

	if err := rBlock.Verify(); err != nil {
		t.Fatalf("expect valid:%v\n", err)
	}
}

func TestBlock(t *testing.T) {
	bp := NewBlockParams(false)
	block := GenBlockFromParams(bp)
	blockBytes := block.Marshal()

	rBlock, err := UnmarshalBlock(bytes.NewReader(blockBytes))
	if err != nil {
		t.Fatalf("unmarshal block failed:%v", err)
	}

	if err := utils.TCheckInt64("time", block.Time, rBlock.Time); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlock(rBlock, bp); err != nil {
		t.Fatal(err)
	}

	if err := rBlock.Verify(); err != nil {
		t.Fatalf("expect valid:%v\n", err)
	}
}

func TestSyncRequest(t *testing.T) {
	base := utils.Hash([]byte("base"))

	req := NewSyncRequest(base)
	reqBytes := req.Marshal()

	rReq, err := UnmarshalSyncRequest(bytes.NewReader(reqBytes))
	if err != nil {
		t.Fatalf("unmarshal SyncRequest failed:%v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgSyncReq, rReq.Type); err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("base", base, rReq.Base); err != nil {
		t.Fatal(err)
	}
}

func TestSyncResponse(t *testing.T) {
	base := utils.Hash([]byte("base"))
	end := utils.Hash([]byte("end"))
	heightDiff := uint32(128)

	resp := NewSyncResponse(base, end, heightDiff, true)
	respBytes := resp.Marshal()

	rResp, err := UnmarshalSyncResponse(bytes.NewReader(respBytes))
	if err != nil {
		t.Fatalf("unmarshal SyncResponse failed:%v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgSyncResp, rResp.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("base", base, rResp.Base); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("end", end, rResp.End); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint32("height diff", heightDiff, resp.HeightDiff); err != nil {
		t.Fatal(err)
	}
	if !rResp.IsUptodate() {
		t.Fatalf("expect uptodate\n")
	}
}

func TestBlockRequest(t *testing.T) {
	base := utils.Hash([]byte("block_base"))
	end := utils.Hash([]byte("block_end"))

	req := NewBlockRequest(base, end, true)
	reqBytes := req.Marshal()

	rReq, err := UnmarshalBlockRequest(bytes.NewReader(reqBytes))
	if err != nil {
		t.Fatalf("unmarshal BlockRequest failed:%v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgBlockRequest, rReq.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("base", base, rReq.Base); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("end", end, rReq.End); err != nil {
		t.Fatal(err)
	}
	if !rReq.IsOnlyHeader() {
		t.Fatalf("expect only header\n")
	}
}

func TestBlockResponse(t *testing.T) {
	blockAParams := NewBlockParams(true)
	blockA := GenBlockFromParams(blockAParams)

	blockBParams := NewBlockParams(false)
	blockB := GenBlockFromParams(blockBParams)

	// response
	resp := NewBlockResponse([]*Block{blockA, blockB})
	respBytes := resp.Marshal()

	rResp, err := UnmarshalBlockResponse(bytes.NewReader(respBytes))
	if err != nil {
		t.Fatalf("unmrshal BlockResponse failed:%v\n", err)
	}

	if utils.TCheckUint8("type", MsgBlockResponse, rResp.Type); err != nil {
		t.Fatal(err)
	}
	if utils.TCheckInt("block num", 2, len(rResp.Blocks)); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlock(rResp.Blocks[0], blockAParams); err != nil {
		t.Fatal(err)
	}
	if err := CheckBlock(rResp.Blocks[1], blockBParams); err != nil {
		t.Fatal(err)
	}
}

func TestBlockBroadcast(t *testing.T) {
	blockParams := NewBlockParams(false)
	block := GenBlockFromParams(blockParams)

	broadcast := NewBlockBroadcast(block)
	broadcastBytes := broadcast.Marshal()

	rBroadcast, err := UnmarshalBlockBroadcast(bytes.NewReader(broadcastBytes))
	if err != nil {
		t.Fatalf("unmarshal block broadcast failed:%v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgBlockBroadcast, rBroadcast.Type); err != nil {
		t.Fatal(err)
	}
	if err := CheckBlock(rBroadcast.Block, blockParams); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceBroadcast(t *testing.T) {
	evdsNum := rand.Intn(10) + 1 // at least one evidence

	var evdsParams []*EvidenceParams
	var evds []*Evidence
	for i := 0; i < evdsNum; i++ {
		params := NewEvidenceParams()
		evdsParams = append(evdsParams, params)
		evds = append(evds, GenEvidenceFromParams(params))
	}

	broadcast := NewEvidenceBroadcast(evds)
	broadcastBytes := broadcast.Marshal()

	rBroadcast, err := UnmarshalEvidenceBroadcast(bytes.NewReader(broadcastBytes))
	if err != nil {
		t.Fatalf("unmarshal evidence broadcast failed:%v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgEvidenceBroadcast, rBroadcast.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("evidence number", evdsNum, len(rBroadcast.Evds)); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < evdsNum; i++ {
		if err := CheckEvidence(rBroadcast.Evds[i], evdsParams[i]); err != nil {
			t.Fatal(err)
		}
	}
}

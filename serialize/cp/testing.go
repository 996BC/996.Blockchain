package cp

// testing.go contains some test helpers

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func errorf(prefix string, expect interface{}, result interface{}) error {
	return fmt.Errorf("%s verify failed, expect %v, result %v", prefix, expect, result)
}

type EvidenceParams struct {
	hash        []byte
	description []byte
	privKey     *btcec.PrivateKey
	pubKeyBytes []byte
	nonce       uint32
}

func NewEvidenceParams() *EvidenceParams {
	hash := utils.Hash(randBytes())
	description := randBytes()
	privKey, _ := btcec.NewPrivateKey(btcec.S256())
	pubKey := privKey.PubKey()

	return &EvidenceParams{
		hash:        hash,
		description: description,
		privKey:     privKey,
		pubKeyBytes: pubKey.SerializeCompressed(),
		nonce:       randNum(),
	}
}

func GenEvidenceFromParams(param *EvidenceParams) *Evidence {
	evd := NewEvidenceV1(param.hash, param.description, param.pubKeyBytes)
	evd.SetNonce(param.nonce)
	evd.Sign(param.privKey)
	return evd
}

func CheckEvidence(e *Evidence, ep *EvidenceParams) error {
	if e.Version != CoreProtocolV1 {
		return errorf("evidence version", CoreProtocolV1, e.Version)
	}
	if !bytes.Equal(e.Hash, ep.hash) {
		return errorf("evidence hash", ep.hash, e.Hash)
	}
	if !bytes.Equal(e.Description, ep.description) {
		return errorf("evidence description", ep.description, e.Description)
	}
	if !bytes.Equal(e.PubKey, ep.pubKeyBytes) {
		return errorf("evidence public key", ep.pubKeyBytes, e.PubKey)
	}
	if e.Nonce != ep.nonce {
		return errorf("evidence nonce", ep.nonce, e.Nonce)
	}

	return nil
}

type BlockHeaderParams struct {
	lastHash []byte
	miner    []byte
	evRoot   []byte
	target   uint32
	nonce    uint32
}

func NewBlockHeaderParams() *BlockHeaderParams {
	minerPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	miner := minerPrivKey.PubKey().SerializeCompressed()

	return &BlockHeaderParams{
		lastHash: utils.Hash(randBytes()),
		miner:    miner,
		evRoot:   utils.Hash(randBytes()),
		target:   randNum(),
		nonce:    randNum(),
	}
}

func GenBlockHeaderFromParams(param *BlockHeaderParams) *BlockHeader {
	blockHeader := NewBlockHeaderV1(param.lastHash, param.miner, param.evRoot)
	blockHeader.SetNonce(param.nonce)
	blockHeader.SetTarget(param.target)
	return blockHeader
}

func CheckBlockHeader(b *BlockHeader, bp *BlockHeaderParams) error {
	if b.Version != CoreProtocolV1 {
		return errorf("block version", CoreProtocolV1, b.Version)
	}
	if b.Nonce != bp.nonce {
		return errorf("block nonce", bp.nonce, b.Nonce)
	}
	if b.Target != bp.target {
		return errorf("block target", bp.target, b.Target)
	}
	if !bytes.Equal(b.LastHash, bp.lastHash) {
		return errorf("block last hash", bp.lastHash, b.LastHash)
	}
	if !bytes.Equal(b.Miner, bp.miner) {
		return errorf("block miner", bp.miner, b.Miner)
	}
	if !bytes.Equal(b.EvidenceRoot, bp.evRoot) {
		return errorf("block evidence root", bp.evRoot, b.EvidenceRoot)
	}

	return nil
}

type BlockParams struct {
	*BlockHeaderParams
	EvdsParams []*EvidenceParams
}

func NewBlockParams(empty bool) *BlockParams {
	headerParams := NewBlockHeaderParams()

	var evdsParams []*EvidenceParams
	if !empty {
		evidenceNum := rand.Intn(10) + 1 // at least one evidence
		for i := 0; i < evidenceNum; i++ {
			evdsParams = append(evdsParams, NewEvidenceParams())
		}
	} else {
		headerParams.evRoot = EmptyEvidenceRoot
	}

	return &BlockParams{
		BlockHeaderParams: headerParams,
		EvdsParams:        evdsParams,
	}
}

func GenBlockFromParams(bp *BlockParams) *Block {
	blockHeader := GenBlockHeaderFromParams(bp.BlockHeaderParams)

	evds := []*Evidence{}
	for _, param := range bp.EvdsParams {
		evds = append(evds, GenEvidenceFromParams(param))
	}

	return NewBlock(blockHeader, evds)
}

func CheckBlock(b *Block, bp *BlockParams) error {
	if err := CheckBlockHeader(b.BlockHeader, bp.BlockHeaderParams); err != nil {
		return err
	}

	if len(b.Evds) != len(bp.EvdsParams) {
		return errorf("evidence size", len(bp.EvdsParams), len(b.Evds))
	}
	for i := 0; i < len(bp.EvdsParams); i++ {
		if err := CheckEvidence(b.Evds[i], bp.EvdsParams[i]); err != nil {
			return nil
		}
	}

	return nil
}

func randBytes() []byte {
	// copy from https://golang.org/pkg/math/rand/#Rand Example
	strs := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
	}
	return []byte(fmt.Sprintf("%s -- %d",
		strs[rand.Intn(len(strs))],
		time.Now().UnixNano()))
}

func randNum() uint32 {
	return rand.Uint32()
}

package blockchain

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/996BC/996.Blockchain/core/merkle"
	"github.com/996BC/996.Blockchain/db"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

type branch struct {
	head *block
	tail *block

	evidenceCache sync.Map // <string, *cp.Evidence>, hex(hash) of evidence as key
	blockCache    sync.Map // <string, *block>
}

func newBranch(begin *block) *branch {
	result := &branch{
		head: begin,
		tail: begin,
	}

	iter := begin
	for {
		bKey := utils.ToHex(iter.hash)
		result.blockCache.Store(bKey, iter)

		for _, e := range iter.Evds {
			eKey := utils.ToHex(e.Hash)
			result.evidenceCache.Store(eKey, e)
		}

		iter = iter.backward
		if iter == nil {
			break
		}
	}

	return result
}

// remove the blocks that have no forward block in the branch
// the branch should never use after removing
// eg.
// A --> B --> C --> D           (main branch)
//             | --> E --> F     (fork branch)
// if the fork branch call remove(), then the C will not point to E, the E will not point to F,
// but the C still points to D
func (b *branch) remove() {
	iter := b.head
	var err error
	for {
		iter, err = iter.remove()
		if err != nil {
			break
		}

		if iter == nil {
			break
		}
	}
}

func (b *branch) add(newBlock *block) error {
	oldHead := b.head
	oldHead.addFordward(newBlock)

	newBlock.setBackward(oldHead)
	nbKey := utils.ToHex(newBlock.hash)
	b.head = newBlock
	b.blockCache.Store(nbKey, newBlock)

	for _, e := range newBlock.Evds {
		key := utils.ToHex(e.Hash)
		b.evidenceCache.Store(key, e)
	}

	return nil
}

func (b *branch) hash() []byte {
	return b.head.hash
}

func (b *branch) height() uint64 {
	return b.head.height
}

func (b *branch) nextBlockTarget(newBlockTime int64) uint32 {
	if b.head.height <= ReferenceBlocks+1 { // ignore the genesis block
		return BlockTargetLimit
	}

	newTime := time.Unix(newBlockTime, 0)
	headTime := time.Unix(b.head.time(), 0)

	preBlock := b.head
	for i := 0; i < ReferenceBlocks; i++ {
		if preBlock.backward == nil {
			logger.Fatal("bug: i %d, b.head.heigh %d, branch info %s\n", i, b.head.height, b.String())
		}
		preBlock = preBlock.backward
	}
	tailBlockTime := time.Unix(preBlock.time(), 0)

	lastTarget := b.head.target()
	return CalculateTarget(lastTarget, newTime.Sub(headTime),
		headTime.Sub(tailBlockTime))
}

func (b *branch) getBlock(hash []byte) *block {
	bKey := utils.ToHex(hash)
	v, ok := b.blockCache.Load(bKey)
	if ok {
		b := v.(*block)
		return b
	}

	return nil
}

func (b *branch) getEvidence(hash []byte) *cp.Evidence {
	eKey := utils.ToHex(hash)
	v, ok := b.evidenceCache.Load(eKey)
	if ok {
		e := v.(*cp.Evidence)
		return e
	}
	return nil
}

func (b *branch) verifyBlock(cb *cp.Block) error {
	// basically check via block itself
	if err := cb.Verify(); err != nil {
		return fmt.Errorf("block struct verify failed:%v", err)
	}

	// deeply check via blockchain context
	// 1. time
	t := time.Unix(cb.Time, 0)
	if t.Sub(time.Now()) > 3*time.Second {
		return fmt.Errorf("invalid future time")
	}
	if t.Before(time.Unix(b.head.time(), 0)) {
		return fmt.Errorf("invalid past time")
	}

	// 2. target
	expectedTarget := b.nextBlockTarget(cb.Time)
	if cb.Target != expectedTarget {
		return fmt.Errorf("mismatch target %d, expect %d", cb.Target, expectedTarget)
	}

	// 3. last block hash
	if !bytes.Equal(b.hash(), cb.LastHash) {
		return fmt.Errorf("mismatch last hash")
	}

	// 4. pow
	if !b.powCheck(TargetToDiff(expectedTarget), cb.GetPow()) {
		return fmt.Errorf("pow check failed")
	}

	// 5. evidence
	if cb.IsEmptyEvidenceRoot() {
		return nil
	}

	var leafs merkle.MerkleLeafs
	for _, e := range cb.Evds {
		if err := b.verifyEvidence(e); err != nil {
			return err
		}
		leafs = append(leafs, e.GetSerializedHash())
	}

	root, _ := merkle.ComputeRoot(leafs)
	if !bytes.Equal(root, cb.EvidenceRoot) {
		return fmt.Errorf("mismatch merkle root")
	}

	return nil
}

// verify
// 1. pow
// 2. whether the account has uploaded the evidence before
func (b *branch) verifyEvidence(e *cp.Evidence) error {
	// basically check via evidence itself
	if err := e.Verify(); err != nil {
		return fmt.Errorf("evidence struct verify failed:%v", err)
	}

	// deeply check via blockchain context
	// pow
	if !b.powCheck(EvidenceDifficultyLimit, e.GetPow()) {
		return fmt.Errorf("pow check of %X failed", e.Hash)
	}

	// cache checking
	if v := b.getEvidence(e.Hash); v != nil {
		return ErrEvidenceAlreadyExist{e.Hash}
	}

	// db checking
	if db.HasEvidence(e.Hash) {
		return ErrEvidenceAlreadyExist{e.Hash}
	}

	return nil
}

func (b *branch) powCheck(limit *big.Int, pow *big.Int) bool {
	return pow.Cmp(limit) == -1
}

func (b *branch) removeFromCache(rmBlock *block) {
	bKey := utils.ToHex(rmBlock.hash)
	b.blockCache.Delete(bKey)

	for _, evd := range rmBlock.Block.Evds {
		eKey := utils.ToHex(evd.Hash)
		b.evidenceCache.Delete(eKey)
	}
}

func (b *branch) String() string {
	var result string
	iter := b.head
	for {
		if iter == nil {
			break
		}
		result += fmt.Sprintf("%X(%d)->",
			iter.hash[utils.HashLength-2:], iter.height)
		iter = iter.backward
	}
	result += "..."

	blocksSize := 0
	b.blockCache.Range(func(key interface{}, value interface{}) bool {
		blocksSize++
		return true
	})
	evdsSize := 0
	b.evidenceCache.Range(func(key interface{}, value interface{}) bool {
		evdsSize++
		return true
	})
	cacheInfo := fmt.Sprintf("cache info: %d blocks, %d evidences", blocksSize, evdsSize)
	result = fmt.Sprintf("%s\n%s", result, cacheInfo)

	return result
}

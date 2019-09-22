package blockchain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

var branchTestVar = &struct {
	blocksNum int

	blocks []*block
	branch *branch // the main branch

	forkIndex  int
	forkBlock  *block
	forkBlocks []*block
	forkBranch *branch
}{
	blocksNum: 4,
}

// generate two branches looks like
// A -> B -> C -> D
//  ......   | -> E (fork from C)
func init() {
	tv := branchTestVar

	for i := 0; i < tv.blocksNum; i++ {
		block := genBlock(uint64(i) + 1)
		tv.blocks = append(tv.blocks, block)
	}

	tv.branch = newBranch(tv.blocks[0])
	for i := 1; i < tv.blocksNum; i++ {
		tv.branch.add(tv.blocks[i])
	}

	// fork from the second last block
	for i := 0; i < tv.blocksNum-1; i++ {
		tv.forkBlocks = append(tv.forkBlocks, tv.blocks[i])
	}
	tv.forkBlock = genBlock(uint64(tv.blocksNum)) // the block E
	tv.forkBlocks = append(tv.forkBlocks, tv.forkBlock)

	tv.forkIndex = tv.blocksNum - 2
	tv.forkBranch = newBranch(tv.blocks[tv.forkIndex]) // fork from the block C
	tv.forkBranch.add(tv.forkBlock)
}

func TestMainBranch(t *testing.T) {
	if err := checkMainBranch(); err != nil {
		t.Fatal(err)
	}
}

func TestForkBranch(t *testing.T) {
	if err := checkForkBranch(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveMainBranch(t *testing.T) {
	tv := branchTestVar
	tv.branch.remove()

	if err := checkForkBranch(); err != nil {
		t.Fatal(err)
	}

	// check the fork point
	forkPoint := tv.blocks[tv.forkIndex]
	if err := utils.TCheckInt("fork point forward block amount", 1, forkPoint.forwardNum()); err != nil {
		t.Fatal(err)
	}

	// recover the main branch
	tv.branch = newBranch(tv.blocks[0])
	for i := 1; i < tv.blocksNum; i++ {
		tv.branch.add(tv.blocks[i])
	}
}

func TestRemoveForkBranch(t *testing.T) {
	tv := branchTestVar
	tv.forkBranch.remove()

	if err := checkMainBranch(); err != nil {
		t.Fatal(err)
	}

	// check the fork point
	forkPoint := tv.blocks[tv.forkIndex]
	if err := utils.TCheckInt("fork point forward block amount", 1, forkPoint.forwardNum()); err != nil {
		t.Fatal(err)
	}

	// recover the fork branch
	tv.forkBranch = newBranch(tv.blocks[tv.forkIndex])
	tv.forkBranch.add(tv.forkBlock)
}

func checkMainBranch() error {
	tv := branchTestVar

	if err := checkHeadTail(tv.branch, tv.blocks[tv.blocksNum-1], tv.blocks[0]); err != nil {
		return err
	}

	if err := checkBlockAndEvidence(tv.branch, tv.blocks, tv.forkBlocks[tv.blocksNum-1:]); err != nil {
		return err
	}

	for i := 0; i < tv.blocksNum-1; i++ {
		if err := checkBlockConnection(tv.blocks[i], tv.blocks[i+1]); err != nil {
			return fmt.Errorf("block %d connection err:%v", i, err)
		}
	}

	return nil
}

func checkForkBranch() error {
	tv := branchTestVar

	if err := checkHeadTail(tv.forkBranch, tv.forkBlock, tv.blocks[tv.forkIndex]); err != nil {
		return err
	}

	if err := checkBlockAndEvidence(tv.forkBranch, tv.forkBlocks, tv.blocks[tv.forkIndex+1:]); err != nil {
		return err
	}

	for i := 0; i < tv.blocksNum-1; i++ {
		if err := checkBlockConnection(tv.forkBlocks[i], tv.forkBlocks[i+1]); err != nil {
			return fmt.Errorf("block %d connection err:%v", i, err)
		}
	}

	return nil
}

func checkHeadTail(b *branch, head *block, tail *block) error {
	if b.head != head {
		return errors.New("branch head mismatch")
	}
	if b.tail != tail {
		return errors.New("branch tail mismatch")
	}

	return nil
}

func checkBlockAndEvidence(b *branch, include []*block, exclude []*block) error {
	for i, block := range include {
		blockHash := block.GetSerializedHash()
		if getResult := b.getBlock(blockHash); getResult == nil {
			return fmt.Errorf("not found block %d in branch", i)
		}

		for j, evd := range block.Evds {
			if getResult := b.getEvidence(evd.Hash); getResult == nil {
				return fmt.Errorf("not found evidence %d:%d in branch", i, j)
			}
		}
	}

	for i, block := range exclude {
		blockHash := block.GetSerializedHash()
		if getResult := b.getBlock(blockHash); getResult != nil {
			return fmt.Errorf("found block %d in branch", i)
		}

		for j, evd := range block.Evds {
			if getResult := b.getEvidence(evd.Hash); getResult != nil {
				return fmt.Errorf("found evidence %d:%d in branch", i, j)
			}
		}
	}

	return nil
}

func checkBlockConnection(parent *block, child *block) error {
	if !parent.isBackwardOf(child.Block) {
		return errors.New("parent forward connection broken")
	}

	if child.backward != parent {
		return errors.New("child backward connection broken")
	}

	return nil
}

func genBlock(height uint64) *block {
	cb := cp.GenBlockFromParams(cp.NewBlockParams(false))
	return newBlock(cb, height, false)
}

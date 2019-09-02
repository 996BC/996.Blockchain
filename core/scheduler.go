package core

import (
	"math/big"

	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/core/merkle"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

// scheduler shcedules mining round by round,
// it collects infomation needed for mining from evidence pool and existed blockchain,
// and broadcast the block once it found the nonce
type scheduler struct {
	pool    *evidencePool
	chain   *blockchain.Chain
	network *net
	pm      *parallelMine
	minerID []byte //compressed public key

	lm *utils.LoopMode
}

func newScheduler(p *evidencePool, c *blockchain.Chain,
	n *net, minerID []byte, parallel int) *scheduler {
	s := &scheduler{
		pool:    p,
		chain:   c,
		network: n,
		pm:      newParallelMine(parallel),
		minerID: minerID,
		lm:      utils.NewLoop(1),
	}

	return s
}

func (s *scheduler) start() {
	go s.schedule()
	s.lm.StartWorking()
}

func (s *scheduler) stop() {
	s.lm.Stop()
}

func (s *scheduler) schedule() {
	s.lm.Add()
	defer s.lm.Done()

	select {
	case <-s.network.InitFinishC:
		logger.Info("blocks sync finished, start mining...")
	case <-s.lm.D:
		logger.Info("program is terminated before blocks sync finished")
		return
	}

	newRound := true
	var lastHash []byte
	var evs []*cp.Evidence
	var block *cp.Block
	var difficulty *big.Int
	for {
		if newRound {
			lastHash = s.chain.LatestBlockHash()
			evs = s.getEvidence()
			block = s.genBlock(evs, lastHash)
		}
		// calculate target/difficulty
		block.SetTarget(s.chain.NextBlockTarget(block.Time))
		difficulty = blockchain.TargetToDiff(block.Target)

		job := s.pm.mine(difficulty, block.BlockHeader)
		logger.Debug("start mining for %v, difficulty %s\n", block.BlockHeader,
			utils.ReadableBigInt(difficulty))

		select {
		case <-s.lm.D:
			job.terminate()
			logger.Info("stop scheduling and exist")
			return
		case <-s.chain.PassiveChangeNotify:
			job.terminate()
			logger.Debug("terminate mining, start next turn\n")

			newRound = true
		case result := <-job.result:
			if result.found {
				logger.Debug("mining found nonce: %d\n", result.nonce)
				block.SetNonce(result.nonce)
				s.network.broadcastBlock(block)
				s.chain.AddBlocks([]*cp.Block{block}, true)

				newRound = true
			} else {
				// refresh the block time and try again
				block.UpdateTime()
				newRound = false
			}
		}
	}
}

func (s *scheduler) getEvidence() []*cp.Evidence {
	evds := make(map[string]*cp.Evidence)

	evdSize := 0
	for evdSize < params.BlockSize {
		e := s.pool.nextEvidence()
		if e == nil {
			break
		}

		if err := s.chain.VerifyEvidence(e); err == nil {
			// exclude the same evidence
			evds[utils.ToHex(e.Hash)] = e
			evdSize += e.Size()
		}
	}

	var result []*cp.Evidence
	for _, evd := range evds {
		result = append(result, evd)
	}
	return result
}

func (s *scheduler) genBlock(evds []*cp.Evidence, lastHash []byte) *cp.Block {
	if len(evds) == 0 {
		return s.genEmptyBlock(lastHash)
	}

	var evLeafs merkle.MerkleLeafs
	for _, e := range evds {
		evLeafs = append(evLeafs, e.GetSerializedHash())
	}
	evRoot, _ := merkle.ComputeRoot(evLeafs)

	header := cp.NewBlockHeaderV1(lastHash, s.minerID, evRoot)
	block := cp.NewBlock(header, evds)

	return block
}

func (s *scheduler) genEmptyBlock(lastHash []byte) *cp.Block {
	header := cp.NewBlockHeaderV1(lastHash, s.minerID, cp.EmptyEvidenceRoot)
	block := cp.NewBlock(header, nil)

	return block
}

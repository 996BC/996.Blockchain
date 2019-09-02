package core

import (
	"math"
	"math/big"
	"time"

	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

const timeoutFactor = 10

type mineResult struct {
	found bool
	nonce uint32
}

type mineJob struct {
	result chan *mineResult
	stop   chan bool
}

func (mj *mineJob) terminate() {
	mj.stop <- true
}

type parallelMine struct {
	parallel int
}

func newParallelMine(parallelNum int) *parallelMine {
	return &parallelMine{
		parallel: parallelNum,
	}
}

// mine starts mining parallely and returns a mineJob; caller will get a results from it or terminate it
func (p *parallelMine) mine(difficulty *big.Int, header *cp.BlockHeader) *mineJob {
	indexInterval := uint32(math.MaxUint32 / p.parallel)
	begin := uint32(0)

	powResult := make(chan *mineResult, p.parallel)
	powStop := make(chan bool, p.parallel)

	for i := 0; i < p.parallel; i++ {
		go p.pow(difficulty, header, begin, begin+indexInterval, powResult, powStop)
		begin += indexInterval
	}

	job := &mineJob{
		result: make(chan *mineResult, 1),
		stop:   make(chan bool, 1),
	}

	go func() {
		finished := 0
		timeoutT := time.NewTicker(blockchain.BlockInterval + blockchain.BlockInterval/timeoutFactor)

		// stop all goroutines
		defer func() {
			for i := 0; i < p.parallel; i++ {
				powStop <- true
			}
		}()

		notFound := &mineResult{
			found: false,
		}

		for {
			select {
			// the caller asks to stop mining
			case <-job.stop:
				return

			// timeout
			case <-timeoutT.C:
				logger.Debugln("pow timeout, recalculate the target")
				job.result <- notFound
				return

			// one of the pow goroutine return its result
			case result := <-powResult:
				if result.found {
					job.result <- result
					return
				}

				// try every nonce but failed
				finished++
				if finished == p.parallel {
					job.result <- notFound
					return
				}
			}
		}
	}()

	return job
}

func (p *parallelMine) pow(difficulty *big.Int, header *cp.BlockHeader, begin, end uint32,
	result chan<- *mineResult, stop <-chan bool) {

	defer func() {
		logger.Debug("[%s]pow goroutine exit\n", utils.ReadableBigInt(difficulty))
	}()

	headerCopy := header.ShallowCopy()
	headerCopy.Nonce = begin
	for {
		select {
		case <-stop:
			return
		default:
			if headerCopy.Nonce == end {
				logger.Debug("pow not found between %d - %d\n", begin, end)
				mr := &mineResult{
					found: false,
				}
				result <- mr
				return
			}

			if pv := headerCopy.NextNonce(); pv.Cmp(difficulty) < 0 {
				mr := &mineResult{
					found: true,
					nonce: headerCopy.Nonce,
				}
				result <- mr
				return
			}
		}
	}
}

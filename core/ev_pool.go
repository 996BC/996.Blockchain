package core

import (
	"math/big"
	"sync"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

const evdsCacheSize = 1024

type RawEvidence struct {
	Hash        []byte
	Description []byte
}

type weightedEvidence struct {
	*cp.Evidence
	weight *big.Int
}

type evidencePool struct {
	key       *btcec.PrivateKey
	raws      chan *RawEvidence
	evds      []*weightedEvidence //ascending order
	evdsMutex sync.Mutex
	broadcast chan<- []*cp.Evidence
	lm        *utils.LoopMode
}

func newEvidencePool(key *btcec.PrivateKey) *evidencePool {
	ep := &evidencePool{
		key:  key,
		raws: make(chan *RawEvidence, evdsCacheSize),
		lm:   utils.NewLoop(1),
	}
	return ep
}

func (e *evidencePool) setBroadcastChan(c chan<- []*cp.Evidence) {
	e.broadcast = c
}

func (e *evidencePool) start() {
	go func() {
		e.lm.Add()
		defer e.lm.Done()
		for {
			select {
			case <-e.lm.D:
				return
			case raw := <-e.raws:
				e.calculateRaw(raw)
			}
		}
	}()

	e.lm.StartWorking()
}

func (e *evidencePool) stop() {
	e.lm.Stop()
}

func (e *evidencePool) addRawEvidence(evds []*RawEvidence) {
	for _, evd := range evds {
		select {
		case e.raws <- evd:
		default:
			logger.Warn("evidence raw queue is full, drop raw evidence %X",
				evd.Hash)
		}
	}
}

func (e *evidencePool) addEvidence(evds []*cp.Evidence, fromBroadcast bool) {
	for _, evd := range evds {
		e.insert(&weightedEvidence{evd, evd.GetPow()})
	}

	if !fromBroadcast {
		e.broadcast <- evds
	}
}

// return next evidence if exists, otherwise return nil
func (e *evidencePool) nextEvidence() *cp.Evidence {
	e.evdsMutex.Lock()
	defer e.evdsMutex.Unlock()

	if len(e.evds) == 0 {
		return nil
	}
	result := e.evds[0]
	e.evds = e.evds[1:]
	return result.Evidence
}

func (e *evidencePool) calculateRaw(raw *RawEvidence) {
	pubKey := e.key.PubKey()
	evd := cp.NewEvidenceV1(raw.Hash, []byte(raw.Description), pubKey.SerializeCompressed())
	if err := evd.Sign(e.key); err != nil {
		logger.Warn("sign evidence failed:%v\n", err)
		return
	}

	//pow
	weight := evd.NextNonce()
	for weight.Cmp(blockchain.EvidenceDifficultyLimit) != -1 {
		weight = evd.NextNonce()
	}
	logger.Debug("find nonce %d for evidence %X\n", evd.Nonce, raw)

	e.insert(&weightedEvidence{evd, weight})
	select {
	case e.broadcast <- []*cp.Evidence{evd}:
	default:
		logger.Warn("evidence ask to broadcast failed\n")
	}
}

func (e *evidencePool) insert(we *weightedEvidence) {
	e.evdsMutex.Lock()
	defer e.evdsMutex.Unlock()

	i := e.binarySearchInsertIndex(we.weight)
	if i == -1 {
		e.evds = append(e.evds, we)
		return
	}

	e.evds = append(e.evds, nil)
	copy(e.evds[i+1:], e.evds[i:])
	e.evds[i] = we

	if len(e.evds) > evdsCacheSize {
		e.evds = e.evds[:evdsCacheSize]
	}
}

func (e *evidencePool) binarySearchInsertIndex(target *big.Int) int {
	if len(e.evds) == 0 {
		return 0
	}

	begin := 0
	end := len(e.evds)
	for {
		mid := (begin + end) / 2
		if e.evds[mid].weight.Cmp(target) >= 0 {
			end = mid
		} else {
			begin = mid + 1
		}

		if begin == end {
			break
		}
	}

	// target is smaller than all
	if end == 0 {
		return 0
	}

	// target is larger than all
	if begin == len(e.evds) {
		return -1
	}

	if e.evds[begin].weight.Cmp(target) > 0 {
		return begin
	}
	return begin + 1
}

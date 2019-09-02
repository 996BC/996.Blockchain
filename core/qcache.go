package core

import (
	"strings"
	"sync"
	"time"

	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/db"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

// qCache caches the unstored blocks for query,
// its data is independent from chain's and only readable
type qCache struct {
	c *blockchain.Chain

	// all the map keys are upper case
	blocks          map[string]*BlockInfo
	evidences       map[string]*EvidenceInfo
	accounts        map[string]*AccountInfo
	sortedBlocks    *sortedBlocks
	lastRefreshTime time.Time
	refreshLock     sync.Mutex
}

func newQCache(c *blockchain.Chain) *qCache {
	return &qCache{
		c: c,
	}
}

type BlockInfo struct {
	*cp.Block
	Height    uint64
	BlockHash []byte
}

type EvidenceInfo struct {
	*cp.Evidence
	Height    uint64
	BlockHash []byte
	Time      int64
}

type AccountInfo struct {
	EvdsHash [][]byte
	Score    uint64
}

type sortedBlocks struct {
	blocks []*BlockInfo
	begin  uint64
	end    uint64
}

func (qc *qCache) getBlockViaHash(hash string) *BlockInfo {
	qc.refresh()
	blocks := qc.blocks
	if v, ok := blocks[strings.ToUpper(hash)]; ok {
		return v
	}
	return nil
}

func (qc *qCache) getBlockViaHeight(height uint64) *BlockInfo {
	qc.refresh()
	sbs := qc.sortedBlocks
	if height >= sbs.begin && height <= sbs.end {
		diff := sbs.end - height
		return sbs.blocks[diff]
	}

	cb, hash, err := db.GetBlockViaHeight(height)
	if err != nil {
		return nil
	}

	return &BlockInfo{
		Block:     cb,
		Height:    height,
		BlockHash: hash,
	}
}

func (qc *qCache) getLatestBlock() *BlockInfo {
	qc.refresh()
	sbs := qc.sortedBlocks
	if len(sbs.blocks) != 0 {
		return sbs.blocks[0]
	}

	latestHeight, err := db.GetLatestHeight()
	if err != nil {
		return nil
	}

	latestBlock, hash, err := db.GetBlockViaHeight(latestHeight)
	if err != nil {
		return nil
	}

	return &BlockInfo{
		Block:     latestBlock,
		Height:    latestHeight,
		BlockHash: hash,
	}
}

func (qc *qCache) getEvidence(hexHash []string) []*EvidenceInfo {
	qc.refresh()
	cacheEvds := qc.evidences

	var result []*EvidenceInfo
	for _, hash := range hexHash {
		if e, ok := cacheEvds[strings.ToUpper(hash)]; ok {
			result = append(result, e)
			continue
		}

		// search in db
		h, err := utils.FromHex(hash)
		if err != nil {
			continue
		}

		e, height, err := db.GetEvidenceViaHash(h)
		if err != nil {
			continue
		}

		header, blockHash, err := db.GetHeaderViaHeight(height)
		if err != nil {
			continue
		}

		result = append(result, &EvidenceInfo{
			Evidence:  e,
			Height:    height,
			BlockHash: blockHash,
			Time:      header.Time,
		})
	}
	return result
}

func (qc *qCache) getAccount(id string) ([][]byte, uint64) {
	qc.refresh()
	cacheAccounts := qc.accounts

	accountKeyB := crypto.IDToBytes(id)
	if accountKeyB == nil {
		return nil, 0
	}

	var evds [][]byte
	score := uint64(0)
	if account, ok := cacheAccounts[strings.ToUpper(id)]; ok {
		evds = append(evds, account.EvdsHash...)
		score = account.Score
	}

	// TODO remove the data both exists in the db and cache
	if evdsHash, _, err := db.GetEvidenceViaKey(accountKeyB); err == nil {
		for _, hash := range evdsHash {
			evds = append(evds, hash)
		}
	}

	if dbScore, err := db.GetScoreViaKey(accountKeyB); err == nil {
		score += dbScore
	}

	return evds, score
}

func (qc *qCache) refresh() {
	qc.refreshLock.Lock()
	defer qc.refreshLock.Unlock()

	const refreshInterval = 20 * time.Second
	now := time.Now()
	if now.Sub(qc.lastRefreshTime) > refreshInterval {
		blocks, heights := qc.c.GetUnstoredBlocks()

		latestSortedBlocks := &sortedBlocks{}
		latestAccounts := make(map[string]*AccountInfo)
		latestEvidences := make(map[string]*EvidenceInfo)
		latestBlocks := make(map[string]*BlockInfo)

		if len(blocks) != 0 {
			latestSortedBlocks.end = heights[0]
			latestSortedBlocks.begin = heights[len(heights)-1]
		}

		for i := 0; i < len(blocks); i++ {
			h := blocks[i].GetSerializedHash()

			blockInfo := &BlockInfo{
				Block:     blocks[i],
				Height:    heights[i],
				BlockHash: h,
			}
			latestBlocks[utils.ToHex(h)] = blockInfo
			latestSortedBlocks.blocks = append(latestSortedBlocks.blocks, blockInfo)

			for _, evd := range blocks[i].Evds {
				latestEvidences[utils.ToHex(evd.Hash)] = &EvidenceInfo{
					Evidence:  evd,
					Height:    heights[i],
					BlockHash: h,
					Time:      blocks[i].Time,
				}

				id := crypto.BytesToID(evd.PubKey)
				var account *AccountInfo
				var ok bool
				if account, ok = latestAccounts[id]; !ok {
					account = &AccountInfo{}
					latestAccounts[id] = account
				}
				account.EvdsHash = append(account.EvdsHash, evd.Hash)
			}

			minerID := crypto.BytesToID(blocks[i].Miner)
			var miner *AccountInfo
			var ok bool
			if miner, ok = latestAccounts[minerID]; !ok {
				miner = &AccountInfo{}
				latestAccounts[minerID] = miner
			}
			miner.Score++
		}

		qc.sortedBlocks = latestSortedBlocks
		qc.accounts = latestAccounts
		qc.evidences = latestEvidences
		qc.blocks = latestBlocks
		qc.lastRefreshTime = now
	}

	return
}

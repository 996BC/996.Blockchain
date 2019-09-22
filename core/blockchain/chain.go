package blockchain

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/996BC/996.Blockchain/db"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	syncMaxBlocks uint64 = 128

	// alpha is the height difference used in manipulating the chain
	// 1. if the branch_a is 'alpha' higher than the branch_b, then removes branch_b from cache
	// 2. if the block forwardNum (fordward reference block number) is 1,
	//  and it is 'alpha' lower than the longest branch, then saves it to db and try to remove it from cache
	// 3. if the received block is 'alpha' lower than the branch, it won't be accepted
	alpha = 8
)

var logger = utils.NewLogger("chain")

type Chain struct {
	PassiveChangeNotify chan bool

	oldestBlock   *block
	branches      []*branch
	longestBranch *branch
	lastHeight    uint64
	branchLock    sync.Mutex
	pendingBlocks chan []*cp.Block
	lm            *utils.LoopMode
}

// NewChain returns a chain, should call only once
func NewChain() *Chain {
	return &Chain{
		PassiveChangeNotify: make(chan bool, 1),
		pendingBlocks:       make(chan []*cp.Block, 16),
		lm:                  utils.NewLoop(1),
	}
}

type Config struct {
	BlockTargetLimit    uint32
	EvidenceTargetLimit uint32
	BlockInterval       int
	Genesis             string
}

// Init initializes the chain from db, should call only once
func (c *Chain) Init(conf *Config) error {
	initMiningParams(conf)

	if !db.HasGenesis() {
		logger.Info("chain starts with empty database")
		if err := c.initGenesis(conf.Genesis); err != nil {
			logger.Warn("chain init failed:%v\n", err)
			return err
		}
		return nil
	}

	return c.initFromDB()
}

func (c *Chain) Start() {
	go c.loop()
	c.lm.StartWorking()
}

func (c *Chain) Stop() {
	c.lm.Stop()
}

// AddBlocks appends new blocks to the chain
func (c *Chain) AddBlocks(blocks []*cp.Block, local bool) {
	if local {
		c.addBlocks(blocks, local)
		return
	}
	c.pendingBlocks <- blocks
}

// NextBlockTarget returns next block required target
func (c *Chain) NextBlockTarget(newBlockTime int64) uint32 {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	return c.longestBranch.nextBlockTarget(newBlockTime)
}

// LatestBlockHash returns the longest branch latest block hash
func (c *Chain) LatestBlockHash() []byte {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()
	return c.longestBranch.hash()
}

// GetSyncHash returns the synchronize used block hash and height difference
func (c *Chain) GetSyncHash(base []byte) (end []byte, heightDiff uint32, err error) {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	var hdiff uint32

	// search in the longest branch
	if baseBlock := c.longestBranch.getBlock(base); baseBlock != nil {
		b := c.longestBranch.head
		if bytes.Equal(b.hash, base) {
			return nil, 0, ErrAlreadyUpToDate{base}
		}

		endHash := b.hash
		for {
			if b == nil {
				// flushing cache to db happens during this time
				return nil, 0, ErrFlushingCache{base}
			}
			if bytes.Equal(b.hash, base) {
				break
			}
			hdiff++
			b = b.backward
		}

		return endHash, hdiff, nil
	}

	// search in the db
	_, baseHeight, err := db.GetHeaderViaHash(base)
	if err != nil {
		return nil, 0, ErrHashNotFound{base}
	}

	_, dbLatestHeight, dbLatestHash, err := db.GetLatestHeader()
	if err != nil {
		return nil, 0, err
	}

	if dbLatestHeight-baseHeight >= syncMaxBlocks {
		respHash, _ := db.GetHash(baseHeight + syncMaxBlocks)
		return respHash, uint32(syncMaxBlocks), nil
	}

	hdiff = uint32(dbLatestHeight - baseHeight)
	return dbLatestHash, hdiff, nil
}

// GetSyncBlocks returns the synchronize used blocks
func (c *Chain) GetSyncBlocks(base []byte, end []byte, onlyHeader bool) ([]*cp.Block, error) {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	var result []*cp.Block

	// search in the longest branch
	baseBlock := c.longestBranch.getBlock(base)
	endBlock := c.longestBranch.getBlock(end)
	if baseBlock != nil && endBlock != nil && baseBlock.height < endBlock.height {
		iter := endBlock
		for {
			if iter.height == baseBlock.height {
				// ignore the base block
				break
			}

			if iter == nil {
				// flushing cache to db happens during this time
				return nil, ErrFlushingCache{base}
			}

			result = append([]*cp.Block{iter.Block.ShallowCopy(onlyHeader)}, result...)
			iter = iter.backward
		}
		return result, nil
	}

	if baseBlock == nil {
		logger.Debug("cache not found base, search in db\n")
	} else if endBlock == nil {
		logger.Debug("cache not found end, search in db\n")
	} else {
		return nil, ErrInvalidBlockRange{fmt.Sprintf("block heigh error, base %d, end %d\n",
			baseBlock.height, endBlock.height)}
	}

	// search in the db
	sBaseBlock, baseHeight, _ := db.GetBlockViaHash(base)
	sEndBlock, endHeight, _ := db.GetBlockViaHash(end)
	if sBaseBlock != nil && sEndBlock != nil && baseHeight < endHeight {
		for i := baseHeight + 1; i <= endHeight; i++ {
			sBlock, _, _ := db.GetBlockViaHeight(i)
			result = append(result, sBlock.ShallowCopy(onlyHeader))
		}
		return result, nil
	}

	if sBaseBlock != nil && endBlock != nil {
		return result, ErrFlushingCache{base}
	}

	return nil, ErrHashNotFound{base}
}

// GetSyncBlockHash returns the latest block hash of each branches
func (c *Chain) GetSyncBlockHash() [][]byte {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	var result [][]byte
	for _, bc := range c.branches {
		result = append(result, bc.hash())
	}
	return result
}

// VerifyEvidence verifys the evidence via the matched branch
func (c *Chain) VerifyEvidence(e *cp.Evidence) error {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	if err := c.longestBranch.verifyEvidence(e); err != nil {
		return err
	}

	return nil
}

// GetUnstoredBlocks returns unstored blocks with their height
// the result is sorted by height in decreasing order
func (c *Chain) GetUnstoredBlocks() ([]*cp.Block, []uint64) {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	var blocks []*cp.Block
	var heights []uint64
	iter := c.longestBranch.head
	for {
		if iter == nil {
			break
		}

		if iter.isStored() {
			break
		}

		blocks = append(blocks, iter.Block)
		heights = append(heights, iter.height)
		iter = iter.backward
	}

	return blocks, heights
}

func (c *Chain) initGenesis(genesis string) error {
	var genesisB []byte
	var cb *cp.Block
	var err error

	if genesisB, err = utils.FromHex(genesis); err != nil {
		return err
	}

	if cb, err = cp.UnmarshalBlock(bytes.NewReader(genesisB)); err != nil {
		return err
	}

	if err = db.PutGenesis(cb); err != nil {
		return err
	}

	// the genesis block height is 1
	c.initFirstBranch(newBlock(cb, 1, true))
	return nil
}

func (c *Chain) initFromDB() error {
	var beginHeight uint64 = 1
	lastHeight, err := db.GetLatestHeight()
	if err != nil {
		logger.Warn("get latest height failed:%v\n", err)
		return err
	}
	if lastHeight > ReferenceBlocks {
		beginHeight = lastHeight - ReferenceBlocks // only takes the last 'ReferenceBlocks' blocks into cache
	}

	var blocks []*block
	for height := beginHeight; height <= lastHeight; height++ {
		cb, _, err := db.GetBlockViaHeight(height)
		if err != nil {
			return fmt.Errorf("height %d, broken db data for block", height)
		}

		blocks = append(blocks, newBlock(cb, height, true))
	}

	bc := c.initFirstBranch(blocks[0])
	for i := 1; i < len(blocks); i++ {
		bc.add(blocks[i])
	}
	return nil
}

func (c *Chain) initFirstBranch(b *block) *branch {
	bc := newBranch(b)
	c.oldestBlock = b
	c.branches = append(c.branches, bc)
	c.longestBranch = bc
	c.lastHeight = c.longestBranch.height()
	return bc
}

func (c *Chain) loop() {
	c.lm.Add()
	defer c.lm.Done()

	maintainTicker := time.NewTicker(time.Duration(2) * BlockInterval)
	statusReportTicker := time.NewTicker(BlockInterval / 2)
	for {
		select {
		case <-c.lm.D:
			return
		case <-maintainTicker.C:
			c.maintain()
		case blocks := <-c.pendingBlocks:
			c.addBlocks(blocks, false)
		case <-statusReportTicker.C:
			c.statusReport()
		}
	}
}

// maintain cleans up the chain and flush cache into db
func (c *Chain) maintain() {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	var reservedBranches []*branch
	for _, bc := range c.branches {
		if c.longestBranch.height()-bc.height() > alpha {
			logger.Debug("remove branch %s\n", bc.String())
			bc.remove()
			continue
		}
		reservedBranches = append(reservedBranches, bc)
	}
	c.branches = reservedBranches

	iter := c.oldestBlock
	for {
		if iter.forwardNum() != 1 {
			break
		}

		// no fork from this block
		if c.longestBranch.height()-iter.height > alpha {
			removingBlock := iter
			if !removingBlock.isStored() {
				if err := db.PutBlock(removingBlock.Block, removingBlock.height); err != nil {
					logger.Fatal("store block failed:%v\n", err)
				}
				removingBlock.stored = true
				logger.Debug("store block (height %d)\n", iter.height)
			}

			// iter++
			removingBlock.fordward.Range(func(k, v interface{}) bool {
				vBlock := v.(*block)
				iter = vBlock
				return true
			})

			// don't remove after storing immediately,
			// keep some blocks both exist in cache and db, for conveniently synchronizing
			if c.longestBranch.height()-iter.height > syncMaxBlocks {

				// disconnect removingBlock with iter
				removingBlock.removeForward(iter)
				iter.removeBackward()

				// remove in each branch's cache
				for _, bc := range c.branches {
					bc.removeFromCache(removingBlock)
				}

				c.oldestBlock = iter
			}

		} else {
			break
		}
	}
}

func (c *Chain) addBlocks(blocks []*cp.Block, local bool) {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	if len(blocks) == 0 {
		logger.Warnln("add blocks failed:empyth blocks")
		return
	}

	var err error
	var bc *branch
	lastHash := blocks[0].LastHash
	bc = c.getBranch(lastHash)
	if bc == nil {
		if bc, err = c.createBranch(blocks[0]); err != nil {
			logger.Info("add blocks failed:%v\n", err)
			return
		}
	}

	for _, cb := range blocks {
		if err := bc.verifyBlock(cb); err != nil {
			logger.Warn("verify blocks failed:%v\n", err)
			return
		}
		bc.add(newBlock(cb, bc.height()+1, false))
	}

	if !local {
		c.notifyCheck()
	}
}

func (c *Chain) getBranch(blochHash []byte) *branch {
	for _, b := range c.branches {
		if bytes.Equal(b.hash(), blochHash) {
			return b
		}
	}
	return nil
}

func (c *Chain) createBranch(newBlock *cp.Block) (*branch, error) {
	var result *branch
	lastHash := newBlock.LastHash

	for _, b := range c.branches {
		if matchBlock := b.getBlock(lastHash); matchBlock != nil {
			if b.height()-matchBlock.height > alpha {
				return nil, fmt.Errorf("the block is too old, branch height %d, block height %d",
					b.height(), matchBlock.height)
			}

			if matchBlock.isBackwardOf(newBlock) {
				return nil, fmt.Errorf("duplicated new block")
			}

			logger.Info("branch fork happen at block %s height %d\n",
				utils.ToHex(matchBlock.hash), matchBlock.height)

			result = newBranch(matchBlock)
			c.branches = append(c.branches, result)
			return result, nil
		}
	}

	return nil, fmt.Errorf("not found branch for last hash %X", lastHash)
}

func (c *Chain) getLongestBranch() *branch {
	var longestBranch *branch
	var height uint64
	for _, b := range c.branches {
		if b.height() > height {
			longestBranch = b
			height = b.height()
		} else if b.height() == height {
			//pick the random one
			if time.Now().Unix()%2 == 0 {
				longestBranch = b
			}
		}
	}
	return longestBranch
}

func (c *Chain) notifyCheck() {
	longestBranch := c.getLongestBranch()
	if longestBranch.height() > c.lastHeight {
		c.longestBranch = longestBranch
		c.lastHeight = c.longestBranch.height()

		select {
		case c.PassiveChangeNotify <- true:
		default:
		}
	}
}

func (c *Chain) statusReport() {
	if utils.GetLogLevel() < utils.LogDebugLevel {
		return
	}

	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	branchNum := len(c.branches)
	text := "\n\toldest: %X with height %d \n\tlongest head:%X \n\tbranch number:%d, details:\n%s"

	var details string
	for i := 0; i < branchNum; i++ {
		details += c.branches[i].String() + "\n\n"
	}

	logger.Debug(text, c.oldestBlock.hash[utils.HashLength-2:], c.oldestBlock.height,
		c.longestBranch.hash()[utils.HashLength-2:], branchNum, details)
}

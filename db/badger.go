package db

import (
	"bytes"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/serialize/storage"
	"github.com/996BC/996.Blockchain/utils"
)

var placeHolder = []byte("0")

type badgerDB struct {
	*badger.DB
	lm *utils.LoopMode
}

func newBadger() *badgerDB {
	return &badgerDB{
		lm: utils.NewLoop(1),
	}
}

func (b *badgerDB) Init(path string) error {
	var dbpath string
	var err error

	if dbpath, err = filepath.Abs(path); err != nil {
		return err
	}

	if err = utils.AccessCheck(dbpath); err != nil {
		return err
	}

	opts := badger.DefaultOptions(dbpath)
	opts = opts.WithLogger(nil)
	opts = opts.WithValueLogFileSize(512 << 20)
	opts = opts.WithMaxTableSize(32 << 20)

	b.DB, err = badger.Open(opts)
	if err != nil {
		return b.wrapError(err)
	}

	b.start()
	return nil
}

func (b *badgerDB) Close() {
	b.stop()
	b.DB.Close()
}

func (b *badgerDB) HasGenesis() bool {
	rf := func(tx *badger.Txn) error {
		_, err := tx.Get(mGenesis)
		return err
	}

	err := b.View(rf)
	if err == nil {
		return true
	} else if err == badger.ErrKeyNotFound {
		return false
	} else {
		logger.Fatal("check genesis failed:%v\n", err)
		return false
	}
}

func (b *badgerDB) PutGenesis(block *cp.Block) error {
	wf := func(tx *badger.Txn) error {
		if err := tx.Set(mGenesis, placeHolder); err != nil {
			return err
		}

		if err := b.putBlockTX(block, 1, tx); err != nil {
			return err
		}

		if err := b.updateLatestHeightTX(1, tx); err != nil {
			return err
		}

		return nil
	}

	return b.update(wf)
}

// PutBlock stores block into db
// It should not modify the existed block and the new block's height should be increased by one
func (b *badgerDB) PutBlock(block *cp.Block, height uint64) error {
	latestHeight, err := b.GetLatestHeight()
	if err != nil {
		return err
	}
	expectHeight := latestHeight + 1
	if height != expectHeight {
		return InvalidHeight{height, expectHeight}
	}

	wf := func(tx *badger.Txn) error {
		if err := b.putBlockTX(block, height, tx); err != nil {
			return err
		}

		if err := b.updateLatestHeightTX(height, tx); err != nil {
			return err
		}

		return nil
	}

	return b.update(wf)
}

// GetHash gets block hash via its height
func (b *badgerDB) GetHash(height uint64) ([]byte, error) {
	var result []byte
	hashKey := getHashKey(height)

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(hashKey)
		if err != nil {
			return err
		}

		result, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	}

	return result, b.view(rf)
}

func (b *badgerDB) GetHeaderViaHeight(height uint64) (*cp.BlockHeader, []byte, error) {
	hash, err := b.GetHash(height)
	if err != nil {
		return nil, nil, err
	}

	header, err := b.getHeader(height, hash)
	if err != nil {
		return nil, nil, err
	}

	return header.BlockHeader, hash, nil
}

func (b *badgerDB) GetHeaderViaHash(h []byte) (*cp.BlockHeader, uint64, error) {
	height, err := b.getHeaderHeight(h)
	if err != nil {
		return nil, 0, err
	}

	header, err := b.getHeader(height, h)
	if err != nil {
		return nil, 0, err
	}

	return header.BlockHeader, height, nil
}

func (b *badgerDB) GetBlockViaHeight(height uint64) (*cp.Block, []byte, error) {
	hash, err := b.GetHash(height)
	if err != nil {
		return nil, nil, err
	}

	result, err := b.getCPBlock(height, hash)
	if err != nil {
		return nil, nil, err
	}

	return result, hash, err
}

func (b *badgerDB) GetBlockViaHash(h []byte) (*cp.Block, uint64, error) {
	height, err := b.getHeaderHeight(h)
	if err != nil {
		return nil, 0, err
	}

	result, err := b.getCPBlock(height, h)
	if err != nil {
		return nil, 0, err
	}

	return result, height, nil
}

func (b *badgerDB) GetEvidenceViaHash(h []byte) (*cp.Evidence, uint64, error) {
	height, err := b.getEvidenceHeight(h)
	if err != nil {
		return nil, 0, err
	}

	evd, err := b.getEvidence(height, h)
	if err != nil {
		return nil, 0, err
	}

	return evd.Evidence, height, nil
}

func (b *badgerDB) GetEvidenceViaKey(pubKey []byte) ([][]byte, []uint64, error) {
	var evdsHash [][]byte
	var heights []uint64

	rf := func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := getAccountEvidenceKeyPrefix(pubKey)
		prefixLen := len(prefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			k := item.Key()
			hash := make([]byte, len(k)-prefixLen)
			copy(hash, k[prefixLen:])
			evdsHash = append(evdsHash, hash)

			err := item.Value(func(v []byte) error {
				heights = append(heights, byteh(v))
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := b.View(rf); err != nil {
		return nil, nil, b.wrapError(err)
	}

	return evdsHash, heights, nil
}

func (b *badgerDB) HasEvidence(h []byte) bool {
	rf := func(tx *badger.Txn) error {
		key := getEvidenceHeightKey(h)
		_, err := tx.Get(key)
		return err
	}

	err := b.View(rf)
	if err == nil {
		return true
	} else if err == badger.ErrKeyNotFound {
		return false
	} else {
		logger.Warn("check evidence failed:%v\n", err)
		return true
	}
}

func (b *badgerDB) GetScoreViaKey(pubKey []byte) (uint64, error) {
	var result uint64

	rf := func(tx *badger.Txn) error {
		scoreKey := getScoreKey(pubKey)
		item, err := tx.Get(scoreKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	err := b.View(rf)
	if err == badger.ErrKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, b.wrapError(err)
	}
	return result, nil
}

func (b *badgerDB) GetLatestHeight() (uint64, error) {
	var result uint64

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(mLatestHeight)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	return result, b.view(rf)
}

func (b *badgerDB) GetLatestHeader() (*cp.BlockHeader, uint64, []byte, error) {
	lastHeight, err := b.GetLatestHeight()
	if err != nil {
		return nil, 0, nil, err
	}

	header, hash, err := b.GetHeaderViaHeight(lastHeight)
	if header == nil {
		return nil, 0, nil, err
	}

	return header, lastHeight, hash, nil
}

func (b *badgerDB) getHeaderHeight(hash []byte) (uint64, error) {
	var result uint64
	headerHeightKey := getHeaderHeightKey(hash)

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(headerHeightKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	return result, b.view(rf)
}

func (b *badgerDB) getEvidenceHeight(hash []byte) (uint64, error) {
	var result uint64
	evidenceHeightKey := getEvidenceHeightKey(hash)

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(evidenceHeightKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	return result, b.view(rf)
}

func (b *badgerDB) getHeader(height uint64, hash []byte) (*storage.BlockHeader, error) {
	var result *storage.BlockHeader
	headerKey := getHeaderKey(height, hash)

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(headerKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result, err = storage.UnmarshalBlockHeader(bytes.NewReader(val))
			return err
		})
	}

	return result, b.view(rf)
}

func (b *badgerDB) getBlock(height uint64, hash []byte) (*storage.Block, error) {
	var result *storage.Block
	blockKey := getBlockKey(height, hash)

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(blockKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result, err = storage.UnmarshalBlock(bytes.NewReader(val))
			return err
		})
	}

	return result, b.view(rf)
}

func (b *badgerDB) getEvidence(height uint64, hash []byte) (*storage.Evidence, error) {
	var result *storage.Evidence
	evidenceKey := getEvidenceKey(height, hash)

	rf := func(tx *badger.Txn) error {
		item, err := tx.Get(evidenceKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result, err = storage.UnmarshalEvidence(bytes.NewReader(val))
			return err
		})
	}

	return result, b.view(rf)
}

func (b *badgerDB) getCPBlock(height uint64, hash []byte) (*cp.Block, error) {
	header, err := b.getHeader(height, hash)
	if err != nil {
		return nil, err
	}

	var evds []*cp.Evidence

	if !header.BlockHeader.IsEmptyEvidenceRoot() {
		storageBlock, err := b.getBlock(height, hash)
		if err != nil {
			return nil, err
		}

		for _, eHash := range storageBlock.EvdsHash {
			evd, err := b.getEvidence(height, eHash)
			if err != nil {
				return nil, err
			}
			evds = append(evds, evd.Evidence)
		}
	}

	return &cp.Block{
		BlockHeader: header.BlockHeader,
		Evds:        evds,
	}, nil
}

func (b *badgerDB) putBlockTX(block *cp.Block, height uint64, tx *badger.Txn) error {
	hash := block.GetSerializedHash()
	header := storage.NewBlockHeader(block.BlockHeader, height)

	if !block.IsEmptyEvidenceRoot() {
		if err := b.putEvidenceTX(hash, block.Evds, height, tx); err != nil {
			return err
		}
	} else {
		header.SetEmptyEvdsRoot()
	}

	storageData := header.Marshal()

	if err := tx.Set(getHeaderKey(height, hash), storageData); err != nil {
		return err
	}
	if err := tx.Set(getHashKey(height), hash); err != nil {
		return err
	}
	if err := tx.Set(getHeaderHeightKey(hash), hbyte(height)); err != nil {
		return err
	}
	if err := b.updateScoreTX(block.Miner, tx); err != nil {
		return err
	}

	return nil
}

func (b *badgerDB) putEvidenceTX(hash []byte, evds []*cp.Evidence, height uint64, tx *badger.Txn) error {
	var evdsHash [][]byte
	for _, e := range evds {
		storageData := e.Marshal()

		if err := tx.Set(getEvidenceKey(height, e.Hash), storageData); err != nil {
			return err
		}
		if err := tx.Set(getEvidenceHeightKey(e.Hash), hbyte(height)); err != nil {
			return err
		}
		if err := b.updateAccountEvidenceTX(e, height, tx); err != nil {
			return err
		}

		evdsHash = append(evdsHash, e.Hash)
	}

	block := storage.NewBlock(evdsHash)
	storageData := block.Marshal()

	if err := tx.Set(getBlockKey(height, hash), storageData); err != nil {
		return err
	}

	return nil
}

func (b *badgerDB) updateAccountEvidenceTX(evd *cp.Evidence, height uint64, tx *badger.Txn) error {
	accountEvidenceKey := append(getAccountEvidenceKeyPrefix(evd.PubKey), evd.Hash...)
	heightValue := hbyte(height)
	return tx.Set(accountEvidenceKey, heightValue)
}

func (b *badgerDB) updateLatestHeightTX(height uint64, tx *badger.Txn) error {
	if err := tx.Set(mLatestHeight, hbyte(height)); err != nil {
		return err
	}
	return nil
}

func (b *badgerDB) updateScoreTX(pubKey []byte, tx *badger.Txn) error {
	scoreKey := getScoreKey(pubKey)

	item, err := tx.Get(scoreKey)
	if err != nil && err != badger.ErrKeyNotFound {
		return err
	}

	origin := uint64(0)
	if err != badger.ErrKeyNotFound {
		item.Value(func(val []byte) error {
			origin = byteh(val)
			return nil
		})
	}

	origin++
	return tx.Set(scoreKey, hbyte(origin))
}

func (b *badgerDB) view(fn func(txn *badger.Txn) error) error {
	return b.wrapError(b.View(fn))
}

func (b *badgerDB) update(fn func(txn *badger.Txn) error) error {
	return b.wrapError(b.Update(fn))
}

// wrap the error directly get from badger
func (b *badgerDB) wrapError(err error) error {
	if err == nil {
		return nil
	}

	if err == badger.ErrKeyNotFound {
		return &NotFound{}
	}

	logger.Warn("badger got unexpect err:%v\n", err)
	return &InternalError{}
}

func (b *badgerDB) start() {
	go b.gcLoop()
	b.lm.StartWorking()
}

func (b *badgerDB) stop() {
	b.lm.Stop()
}

func (b *badgerDB) gcLoop() {
	b.lm.Add()
	defer b.lm.Done()

	ticker := time.NewTicker(10 * time.Minute)

	for {
		select {
		case <-b.lm.D:
			return
		case <-ticker.C:
			b.RunValueLogGC(0.5)
		}
	}
}

package cp

import (
	"encoding/binary"
	"math/big"

	"github.com/996BC/996.Blockchain/utils"
)

// powCache is used in mining, to decrease the times for calling Marshal();
// the result is only readable, should not modify it
type powCache struct {
	marshalCache []byte
	powCache     *big.Int
	cache        bool
}

func newPowCache() *powCache {
	return &powCache{
		cache: false,
	}
}

func (p *powCache) cacheBefore() bool {
	return p.cache
}

func (p *powCache) setCache(marshal []byte, pow *big.Int) {
	p.marshalCache = marshal
	p.powCache = pow
	p.cache = true
}

func (p *powCache) update(nonce uint32, index int) *big.Int {
	binary.BigEndian.PutUint32(p.marshalCache[index:], nonce)
	return p.powCache.SetBytes(utils.Hash(p.marshalCache))
}

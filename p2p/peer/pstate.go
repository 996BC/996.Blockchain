package peer

import (
	"time"
)

const (
	peerExpiredTime      = 35 * time.Second
	getNeighbourInterval = 15 * time.Second
	pingInterval         = 10 * time.Second
)

var (
	initTimepoint = time.Unix(0, 0)
)

// peer state
type pstate struct {
	*Peer
	isSeed               bool //seeds should never be removed once they were added into peers
	hasPingBefore        bool
	lastActiveTime       time.Time
	lastGetNeighbourTime time.Time
}

func newPState(p *Peer, isSeed bool) *pstate {
	return &pstate{
		Peer:                 p,
		isSeed:               isSeed,
		hasPingBefore:        false,
		lastActiveTime:       initTimepoint,
		lastGetNeighbourTime: initTimepoint,
	}
}

func (p *pstate) isTimeToPing() bool {
	return time.Now().Sub(p.lastActiveTime) >= pingInterval
}

func (p *pstate) isAvaible() bool {
	return time.Now().Sub(p.lastActiveTime) < peerExpiredTime
}

func (p *pstate) isTimeToGetNeighbours() bool {
	return time.Now().Sub(p.lastGetNeighbourTime) >= getNeighbourInterval
}

func (p *pstate) isToRemove() bool {
	if !p.isAvaible() && p.hasPingBefore && !p.isSeed {
		return true
	}
	return false
}

func (p *pstate) doPing() {
	p.hasPingBefore = true
}

func (p *pstate) updateActiveTime() {
	p.lastActiveTime = time.Now()
}

func (p *pstate) updateGetNeighbourTime() {
	p.lastGetNeighbourTime = time.Now()
}

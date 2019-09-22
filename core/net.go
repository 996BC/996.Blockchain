package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/p2p"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	coreProtocolID           = 100
	coreProtocol             = "CoreProtocol"
	maxBlocksNumInResponse   = 16
	initializingSyncInterval = 1 * time.Second
	syncInterval             = 5 * time.Second
)

type waitingBlocks struct {
	peerID           string
	lastResponseTime time.Time
	remainNums       uint32
	response         []*cp.BlockResponse
}

// net runs the "CoreProtocol" with other peers
// to archeive agreement on blockchain through the
// longest chain rule
type net struct {
	InitFinishC chan bool
	inited      bool

	lightNode bool
	pr        p2p.ProtocolRunner
	sendQ     chan *p2p.PeerData
	chain     *blockchain.Chain
	pool      *evidencePool

	syncTicker    *time.Ticker
	syncHashResp  map[string]*cp.SyncResponse // peerID as key
	watingHash    bool
	waitingBlocks []*waitingBlocks

	evdsToBroadcast chan []*cp.Evidence
	broadcastFilter map[string]time.Time
	lm              *utils.LoopMode
}

func newNet(node p2p.Node, chain *blockchain.Chain, pool *evidencePool, nodeType params.NodeType) *net {
	result := &net{
		InitFinishC:     make(chan bool, 1),
		inited:          false,
		lightNode:       nodeType == params.LightNode,
		sendQ:           make(chan *p2p.PeerData, 512),
		chain:           chain,
		pool:            pool,
		syncTicker:      time.NewTicker(initializingSyncInterval),
		syncHashResp:    make(map[string]*cp.SyncResponse),
		watingHash:      false,
		evdsToBroadcast: make(chan []*cp.Evidence, evdsCacheSize),
		broadcastFilter: make(map[string]time.Time),
		lm:              utils.NewLoop(2),
	}

	result.pr = node.AddProtocol(result)
	return result
}

func (n *net) ID() uint8 {
	return coreProtocolID
}

func (n *net) Name() string {
	return coreProtocol
}

func (n *net) start() {
	go n.loop()
	go n.doSend()
	n.lm.StartWorking()
}

func (n *net) stop() {
	n.lm.Stop()
}

func (n *net) loop() {
	n.lm.Add()
	defer n.lm.Done()

	cleanupTicker := time.NewTicker(30 * time.Second)
	recvPktChan := n.pr.GetRecvChan()

	n.sync()
	for {
		select {
		case <-n.lm.D:
			return
		case pkt := <-recvPktChan:
			n.handleRecvPacket(pkt)
		case <-n.syncTicker.C:
			n.sync()
		case evds := <-n.evdsToBroadcast:
			n.broadcastEvidence(evds)
		case <-cleanupTicker.C:
			now := time.Now()

			for k, v := range n.broadcastFilter {
				if now.Sub(v) > 1*time.Hour {
					delete(n.broadcastFilter, k)
				}
			}

		}
	}
}

func (n *net) send(data []byte, peerID string) {
	n.sendQ <- &p2p.PeerData{
		Data: data,
		Peer: peerID,
	}
}

func (n *net) broadcast(data []byte) {
	h := utils.Hash(data)
	encoded := base64.StdEncoding.EncodeToString(h)
	n.broadcastFilter[encoded] = time.Now()

	select {
	case n.sendQ <- &p2p.PeerData{
		Data: data,
	}:
	default:
		logger.Warn("net send queue full, drop packet")
	}
}

func (n *net) doSend() {
	n.lm.Add()
	defer n.lm.Done()

	for {
		select {
		case <-n.lm.D:
			return
		case sendData := <-n.sendQ:
			if err := n.pr.Send(sendData); err != nil {
				logger.Warn("send failed: %v\n", err)
			}
		}
	}
}

func (n *net) handleRecvPacket(pd *p2p.PeerData) {
	var err error
	var msg *cp.Head

	if msg, err = cp.UnmarshalHead(bytes.NewReader(pd.Data)); err != nil {
		return
	}

	errorlog := func() {
		logger.Warn("receive err type(%d) msg from %s\n", msg.Type, pd.Peer)
		return
	}

	// ignore broadcast before init finish
	if !n.inited && (msg.Type == cp.MsgBlockBroadcast || msg.Type == cp.MsgEvidenceBroadcast) {
		return
	}

	data := bytes.NewReader(pd.Data)
	switch msg.Type {
	case cp.MsgSyncReq:
		var syncRequest *cp.SyncRequest
		if syncRequest, err = cp.UnmarshalSyncRequest(data); err != nil {
			errorlog()
			return
		}
		n.handleSyncRequest(syncRequest, pd.Peer)

	case cp.MsgSyncResp:
		var syncHashResp *cp.SyncResponse
		if syncHashResp, err = cp.UnmarshalSyncResponse(data); err != nil {
			errorlog()
			return
		}
		n.handleSyncResponse(syncHashResp, pd.Peer)

	case cp.MsgBlockRequest:
		var blockRequest *cp.BlockRequest
		if blockRequest, err = cp.UnmarshalBlockRequest(data); err != nil {
			errorlog()
			return
		}
		n.handleBlocksRequest(blockRequest, pd.Peer)

	case cp.MsgBlockResponse:
		var blockResponse *cp.BlockResponse
		if blockResponse, err = cp.UnmarshalBlockResponse(data); err != nil {
			errorlog()
			return
		}
		n.handleBlocksResponse(blockResponse, pd.Peer)

	case cp.MsgBlockBroadcast:
		var block *cp.BlockBroadcast
		if block, err = cp.UnmarshalBlockBroadcast(data); err != nil {
			errorlog()
			return
		}
		n.handleBlockBroadcast(pd.Data, block, pd.Peer)

	case cp.MsgEvidenceBroadcast:
		var evds *cp.EvidenceBroadcast
		if evds, err = cp.UnmarshalEvidenceBroadcast(data); err != nil {
			errorlog()
			return
		}
		n.handleEvidenceBroadcase(pd.Data, evds, pd.Peer)

	default:
		errorlog()
	}
}

// sync asks the neighbours to sync blocks in 2 steps:
// step 1. sync the hash of blocks
// step 2. sync the blocks
// so it will take 2 * syncInterval time to finish
func (n *net) sync() {
	// if it is not waiting for blocks， do the sync request
	if len(n.waitingBlocks) == 0 {
		n.syncRequest()
		return
	}

	// otherwise clean up the timeout wating
	now := time.Now()
	var waiting []*waitingBlocks
	for _, exp := range n.waitingBlocks {
		// expect transfering a block per 5 seconds
		if now.Sub(exp.lastResponseTime) <= time.Duration(5*maxBlocksNumInResponse)*time.Second {
			waiting = append(waiting, exp)
			continue
		}

		logger.Info("peer %s response blocks timeout(now %s, last active %s), remain %d\n",
			exp.peerID, utils.TimeToString(now),
			utils.TimeToString(exp.lastResponseTime), exp.remainNums)

	}
	n.waitingBlocks = waiting
}

func (n *net) syncRequest() {
	// if the sync hash response is empty, send sync request for hash
	if len(n.syncHashResp) == 0 {
		latestHash := n.chain.GetSyncBlockHash()
		for _, h := range latestHash {
			request := cp.NewSyncRequest(h).Marshal()
			n.broadcast(request)
		}
		n.watingHash = true
		return
	}

	// otherwise send sync request for blocks
	queryFilter := make(map[string]bool)
	alreadyUptodate := true
	for peerID, resp := range n.syncHashResp {
		if resp.IsUptodate() {
			continue
		}
		alreadyUptodate = false

		// filters the same response
		queryFlag := fmt.Sprintf("%X-%d", resp.End, resp.HeightDiff)
		if _, find := queryFilter[queryFlag]; find {
			continue
		}
		queryFilter[queryFlag] = true

		request := cp.NewBlockRequest(resp.Base, resp.End, n.lightNode).Marshal()
		n.send(request, peerID)

		// add to waiting list
		exp := &waitingBlocks{
			peerID:           peerID,
			lastResponseTime: time.Now(),
			remainNums:       resp.HeightDiff,
		}
		n.waitingBlocks = append(n.waitingBlocks, exp)
		logger.Debug("add block response expection, peer:%s remainNums:%d, from %X to %X\n",
			exp.peerID, exp.remainNums, resp.Base, resp.End)
	}

	// cleanup
	n.syncHashResp = make(map[string]*cp.SyncResponse)
	n.watingHash = false

	// finish initializing
	if !n.inited && alreadyUptodate {
		n.InitFinishC <- true
		n.inited = true
		// reduce the sync request frequency
		n.syncTicker.Stop()
		n.syncTicker = time.NewTicker(syncInterval)
		logger.Debug("network for CoreProtocol init finished")
	}
}

func (n *net) broadcastBlock(b *cp.Block) {
	content := cp.NewBlockBroadcast(b).Marshal()
	n.broadcast(content)
}

func (n *net) broadcastEvidence(evds []*cp.Evidence) {
	content := cp.NewEvidenceBroadcast(evds).Marshal()
	n.broadcast(content)
}

func (n *net) handleSyncRequest(r *cp.SyncRequest, peerID string) {
	logger.Debug("receive SyncRequest from %s, %v\n", peerID, r)
	syncEnd, heightDiff, err := n.chain.GetSyncHash(r.Base)
	var response []byte

	if err != nil {
		if _, ok := err.(blockchain.ErrAlreadyUpToDate); ok {
			response = cp.NewSyncResponse(nil, nil, 0, true).Marshal()
			logger.Debug("reply sync request already uptodate\n")
		} else {
			logger.Debug("%v\n", err)
			return
		}
	} else {
		response = cp.NewSyncResponse(r.Base, syncEnd, heightDiff, false).Marshal()
		logger.Debug("replay %d sync request with block hash\n", heightDiff)
	}

	if response == nil {
		logger.Warn("generate SyncRequest response failed\n")
		return
	}

	n.send(response, peerID)
}

func (n *net) handleSyncResponse(r *cp.SyncResponse, peerID string) {
	if !n.watingHash {
		return
	}

	logger.Debug("receive SyncResponse from %s, %v\n", peerID, r)
	n.syncHashResp[peerID] = r
}

func (n *net) handleBlocksRequest(r *cp.BlockRequest, peerID string) {
	logger.Debug("receive BlockRequest from %s, base %X\n", peerID, r.Base)
	blocks, err := n.chain.GetSyncBlocks(r.Base, r.End, r.IsOnlyHeader())
	if err != nil {
		logger.Warn("%v\n", err)
		return
	}

	logger.Debug("reply BlockRequest with %d blocks\n", len(blocks))
	for len(blocks) > 0 {
		sendNum := maxBlocksNumInResponse
		if len(blocks) < maxBlocksNumInResponse {
			sendNum = len(blocks)
		}

		response := cp.NewBlockResponse(blocks[:sendNum]).Marshal()
		if response == nil {
			logger.Warn("generate BlockResponse failed\n")
			return
		}
		n.send(response, peerID)

		blocks = blocks[sendNum:]
	}
}

func (n *net) handleBlocksResponse(r *cp.BlockResponse, peerID string) {
	logger.Debug("receive BlockResponse from %s, %d blocks\n", peerID, len(r.Blocks))
	for i, exp := range n.waitingBlocks {
		if exp.peerID == peerID {
			exp.lastResponseTime = time.Now()
			exp.remainNums -= uint32(len(r.Blocks))
			exp.response = append(exp.response, r)

			remove := false
			if exp.remainNums == 0 {
				logger.Info("finish blocks sync from %s\n", peerID)

				var toAddBlocks []*cp.Block
				for _, resp := range exp.response {
					toAddBlocks = append(toAddBlocks, resp.Blocks...)
				}
				n.chain.AddBlocks(toAddBlocks, false)
				remove = true
			}
			if exp.remainNums < 0 {
				logger.Warn("receiv err block response from %s\n", peerID)
				remove = true
			}

			if remove {
				n.waitingBlocks = append(n.waitingBlocks[:i], n.waitingBlocks[i+1:]...)
			}

			return
		}
	}

}

func (n *net) handleBlockBroadcast(originData []byte, b *cp.BlockBroadcast, peerID string) {
	if n.relayBroadcast(originData) {
		hash := b.Block.GetSerializedHash()
		logger.Debug("first time receive block broadcast from %s, hash %X\n", peerID, hash)
		n.chain.AddBlocks([]*cp.Block{b.Block}, false)
	}
}

func (n *net) handleEvidenceBroadcase(originData []byte, b *cp.EvidenceBroadcast, peerID string) {
	if n.relayBroadcast(originData) {
		logger.Debug("first time receive evidence broadcast from %s, %v\n", peerID, b)
		n.pool.addEvidence(b.Evds, true)
	}
}

func (n *net) relayBroadcast(originData []byte) bool {
	h := utils.Hash(originData)
	encoded := base64.StdEncoding.EncodeToString(h)
	if _, ok := n.broadcastFilter[encoded]; ok {
		return false
	}

	n.broadcastFilter[encoded] = time.Now()
	n.broadcast(originData)
	return true
}

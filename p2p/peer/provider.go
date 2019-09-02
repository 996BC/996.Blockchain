package peer

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/btcsuite/btcd/btcec"

	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/serialize/discover"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	msgDiscardTime      int64 = 8 //8s
	maxNeighboursRspNum       = 8
	pingHashExpiredTime       = peerExpiredTime
)

var (
	logger = utils.NewLogger("discover")
)

type provider struct {
	ip            net.IP
	port          int
	compressedKey []byte
	udp           utils.UDPServer
	table         table
	pingHash      map[string]time.Time // hash as key

	lm *utils.LoopMode
}

func NewProvider(ipstr string, port int, publicKey *btcec.PublicKey) Provider {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		logger.Fatal("invalid ip:%s\n", ipstr)
	}

	p := &provider{
		ip:            ip,
		port:          port,
		compressedKey: publicKey.SerializeCompressed(),
		table:         newTable(crypto.PubKeyToID(publicKey)),
		pingHash:      make(map[string]time.Time),
		lm:            utils.NewLoop(1),
	}
	p.udp = utils.NewUDPServer(ip, port)

	return p
}

func (p *provider) Start() {
	if !p.udp.Start() {
		logger.Fatalln("start udp server failed")
	}

	go p.loop()
	p.lm.StartWorking()
}

func (p *provider) Stop() {
	if p.lm.Stop() {
		p.udp.Stop()
	}
}

func (p *provider) AddSeeds(seeds []*Peer) {
	p.table.addPeers(seeds, true)
}

func (p *provider) GetPeers(expect int, exclude map[string]bool) ([]*Peer, error) {
	return p.table.getPeers(expect, exclude), nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[provider] id:%s, with %s:%d\n",
		crypto.BytesToID(p.compressedKey), p.ip.String(), p.port)
}

func (p *provider) loop() {
	p.lm.Add()
	defer p.lm.Done()

	refreshTicker := time.NewTicker(peerExpiredTime * 2)
	taskTicker := time.NewTicker(2 * time.Second)
	recvQ := p.udp.GetRecvChannel()

	for {
		select {
		case <-p.lm.D:
			return
		case <-taskTicker.C:
			p.ping()
			p.getNeighbours()
		case pkt := <-recvQ:
			p.handleRecv(pkt)
		case <-refreshTicker.C:
			p.refresh()
		}
	}
}

func (p *provider) handleRecv(pkt *utils.UDPPacket) {
	head, err := discover.UnmarshalHead(bytes.NewReader(pkt.Data))
	if err != nil {
		logger.Warn("receive error data\n")
		return
	}

	now := time.Now().Unix()
	if head.Time+msgDiscardTime < now {
		logger.Debug("expired Packet from %v\n", pkt.Addr)
		return
	}

	switch head.Type {
	case discover.MsgPing:
		p.handlePing(pkt.Data, pkt.Addr)
	case discover.MsgPong:
		p.handlePong(pkt.Data, pkt.Addr)
	case discover.MsgGetNeighbours:
		p.handleGetNeigoubours(pkt.Data, pkt.Addr)
	case discover.MsgNeighbours:
		p.handleNeigoubours(pkt.Data, pkt.Addr)
	default:
		logger.Warn("unknown op:%d\n", head.Type)
		return
	}
}

func (p *provider) send(msg []byte, addr *net.UDPAddr) {
	pkt := &utils.UDPPacket{
		Data: msg,
		Addr: addr,
	}
	p.udp.Send(pkt)
}

func (p *provider) ping() {
	targets := p.table.getPeersToPing()

	for _, peer := range targets {
		pkt := discover.NewPing(p.compressedKey).Marshal()

		if addr, err := net.ResolveUDPAddr("udp", peer.Address()); err == nil {
			p.send(pkt, addr)
			p.pingHash[utils.ToHex(utils.Hash(pkt))] = time.Now()
		}
	}
}

func (p *provider) getNeighbours() {
	targets := p.table.getPeersToGetNeighbours()

	for _, peer := range targets {
		pkt := discover.NewGetNeighbours(p.compressedKey).Marshal()

		if addr, err := net.ResolveUDPAddr("udp", peer.Address()); err == nil {
			p.send(pkt, addr)
		}
	}
}

func (p *provider) handlePing(data []byte, remoteAddr *net.UDPAddr) {
	ping, err := discover.UnmarshalPing(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error ping:%v\n", err)
		return
	}

	key, err := btcec.ParsePubKey(ping.PubKey, btcec.S256())
	if err != nil {
		logger.Warn("parse ping key failed:%v\n", err)
	}
	p.table.recvPing(NewPeer(remoteAddr.IP, remoteAddr.Port, key))

	// response ping
	pingHash := utils.Hash(data)
	pong := discover.NewPong(pingHash, p.compressedKey).Marshal()
	if pong == nil {
		logger.Warn("generate Pong failed\n")
		return
	}
	p.send(pong, remoteAddr)
}

func (p *provider) handlePong(data []byte, remoteAddr *net.UDPAddr) {
	pong, err := discover.UnmarshalPong(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error Pong:%v\n", err)
		return
	}

	pingHash := utils.ToHex(pong.PingHash)
	if _, ok := p.pingHash[pingHash]; !ok {
		return
	}
	delete(p.pingHash, pingHash)

	key, err := btcec.ParsePubKey(pong.PubKey, btcec.S256())
	if err != nil {
		logger.Warn("parse ping key failed:%v\n", err)
	}
	p.table.recvPong(NewPeer(remoteAddr.IP, remoteAddr.Port, key))
}

func (p *provider) handleGetNeigoubours(data []byte, remoteAddr *net.UDPAddr) {
	getNeibours, err := discover.UnmarshalGetNeighbours(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error GetNeighbours:%v\n", err)
		return
	}

	remotePubKey, err := btcec.ParsePubKey(getNeibours.PubKey, btcec.S256())
	if err != nil {
		logger.Warn("parse GetNeighbours PubKey failed:%v\n", err)
	}

	remoteID := crypto.BytesToID(getNeibours.PubKey)
	if !p.table.exists(remoteID) {
		logger.Warn("query is not from my peer and ignore it: %v\n", remoteAddr)
		return
	}

	// response
	exclude := make(map[string]bool)
	exclude[remoteID] = true

	neighbours := p.table.getPeers(maxNeighboursRspNum, exclude)
	neighboursMsg := p.genNeighbours(neighbours)
	p.send(neighboursMsg, remoteAddr)

	// also notify the neighbours about the getter
	putMsg := p.genNeighbours([]*Peer{NewPeer(remoteAddr.IP, remoteAddr.Port, remotePubKey)})
	for _, n := range neighbours {
		if neighbourAddr, err := net.ResolveUDPAddr("udp", n.Address()); err == nil {
			p.send(putMsg, neighbourAddr)
		}
	}
}

func (p *provider) handleNeigoubours(data []byte, remoteAddr *net.UDPAddr) {
	neighbours, err := discover.UnmarshalNeighbours(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error Neighbours:%v\n", err)
		return
	}

	var peers []*Peer
	for _, n := range neighbours.Nodes {
		pubKey, err := btcec.ParsePubKey(n.PubKey, btcec.S256())
		if err != nil {
			logger.Warn("parse Neighbours PubKey failed:%v\n", err)
			continue
		}
		peers = append(peers, NewPeer(n.Addr.IP, int(n.Addr.Port), pubKey))
	}

	p.table.addPeers(peers, false)
}

func (p *provider) refresh() {
	p.table.refresh()

	curr := time.Now()
	for k, v := range p.pingHash {
		if curr.Sub(v) > pingHashExpiredTime {
			delete(p.pingHash, k)
		}
	}
}

func (p *provider) genNeighbours(peers []*Peer) []byte {
	var nodes []*discover.Node
	for _, p := range peers {
		addr := discover.NewAddress(p.IP.String(), int32(p.Port))
		node := discover.NewNode(addr, crypto.IDToBytes(p.ID))
		nodes = append(nodes, node)
	}

	neighbours := discover.NewNeighbours(nodes)
	return neighbours.Marshal()
}

// only used in test
func (p *provider) getAllPeersForTest() map[string]*Peer {
	result := make(map[string]*Peer)
	table := p.table.(*tableImp)

	for _, pst := range table.peers {
		if pst.isAvaible() {
			result[pst.ID] = pst.Peer
		}
	}
	return result
}

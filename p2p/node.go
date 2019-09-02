package p2p

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/utils"
)

var logger = utils.NewLogger("p2p")

// Config is configs for the p2p network Node
type Config struct {
	NodeIP     string
	NodePort   int
	Provider   peer.Provider
	MaxPeerNum int
	PrivKey    *btcec.PrivateKey
	Type       params.NodeType
	ChainID    uint8
}

// Node is a node that can communicate with others in the p2p network
type Node struct {
	tcpServer utils.TCPServer
	privKey   *btcec.PrivateKey
	chainID   uint8
	nodeType  params.NodeType

	maxPeersNum  int
	peerProvider peer.Provider

	connsMutex sync.Mutex
	conns      map[string]*conn //<peer ID, conn>

	protocolsMutex sync.Mutex
	protocols      map[uint8]*protocolRunner //<Protocol ID, ProtocolRunner>

	ng          *negotiator
	ngMutex     sync.Mutex
	ngBlackList map[string]time.Time

	connectTask chan *peer.Peer
	delConnTask chan string
	lm          *utils.LoopMode
}

// NewNode returns a p2p network Node
func NewNode(c *Config) *Node {
	if c.Type != params.FullNode && c.Type != params.LightNode {
		logger.Fatal("invalid node type %d\n", c.Type)
	}
	n := &Node{
		privKey:      c.PrivKey,
		chainID:      c.ChainID,
		nodeType:     c.Type,
		maxPeersNum:  c.MaxPeerNum,
		peerProvider: c.Provider,
		conns:        make(map[string]*conn),
		protocols:    make(map[uint8]*protocolRunner),
		ngBlackList:  make(map[string]time.Time),
		connectTask:  make(chan *peer.Peer, c.MaxPeerNum),
		delConnTask:  make(chan string, c.MaxPeerNum),
		lm:           utils.NewLoop(1),
	}
	n.ng = newNegotiator(n.privKey, n.chainID, n.nodeType)

	var ip net.IP
	if ip = net.ParseIP(c.NodeIP); ip == nil {
		logger.Fatal("parse ip for tcp server failed:%s\n", c.NodeIP)
	}
	n.tcpServer = utils.NewTCPServer(ip, c.NodePort)

	return n
}

func (n *Node) String() string {
	return fmt.Sprintf("[Node] listen on %v", n.tcpServer.Addr())
}

// AddProtocol adds the runtime p2p network protocols
func (n *Node) AddProtocol(p Protocol) ProtocolRunner {
	n.protocolsMutex.Lock()
	defer n.protocolsMutex.Unlock()

	if v, ok := n.protocols[p.ID()]; ok {
		logger.Fatal("protocol conflicts in ID:%s, exists:%s, wanted to add:%s",
			p.ID(), v.protocol.Name(), v.protocol.Name())
	}
	runner := newProtocolRunner(p, n)
	n.protocols[p.ID()] = runner
	return runner
}

func (n *Node) Start() {
	if !n.tcpServer.Start() {
		logger.Fatalln("start node's tcp server failed")
	}

	go n.loop()
	n.lm.StartWorking()
}

func (n *Node) Stop() {
	if n.lm.Stop() {
		n.tcpServer.Stop()
		n.connsMutex.Lock()
		for _, conn := range n.conns {
			conn.stop()
		}
		n.connsMutex.Unlock()
	}
}

func (n *Node) loop() {
	n.lm.Add()
	defer n.lm.Done()

	checkPeersTicker := time.NewTicker(10 * time.Second)
	statusReportTicker := time.NewTicker(15 * time.Second)
	ngBlackListCleanTicker := time.NewTicker(1 * time.Minute)

	acceptConn := n.tcpServer.GetTCPAcceptConnChannel()
	for {
		select {
		case <-n.lm.D:
			return
		case delConnID := <-n.delConnTask:
			n.connsMutex.Lock()
			delete(n.conns, delConnID)
			n.connsMutex.Unlock()
		case <-checkPeersTicker.C:
			n.checkPeers()
		case <-statusReportTicker.C:
			n.statusReport()
		case <-ngBlackListCleanTicker.C:
			n.cleanNgBlackList()
		case newPeer := <-n.connectTask:
			go func() {
				n.lm.Add()
				n.setupConn(newPeer)
				n.lm.Done()
			}()
		case newPeerConn := <-acceptConn:
			go func() {
				n.lm.Add()
				newPeerConn.SetSplitFunc(splitTCPStream)
				n.recvHandshake(newPeerConn)
				n.lm.Done()
			}()
		}
	}
}

func (n *Node) checkPeers() {
	peersNum := len(n.conns)
	if peersNum > n.maxPeersNum {
		return
	}

	expectNum := n.maxPeersNum - peersNum
	excludePeers := n.getExcludePeers()
	newPeers, err := n.peerProvider.GetPeers(expectNum, excludePeers)
	if err != nil {
		logger.Warn("get peers from provider failed:%v\n", err)
		return
	}
	for _, newPeer := range newPeers {
		n.connectTask <- newPeer
	}

}

func (n *Node) statusReport() {
	if utils.GetLogLevel() < utils.LogDebugLevel {
		return
	}

	var peersInfo string
	n.connsMutex.Lock()
	for k, v := range n.conns {
		peersInfo += "[" + k[:6] + " " + v.p.Address() + "] "
	}
	n.connsMutex.Unlock()

	logger.Debug("current address book:%s\n", peersInfo)
}

func (n *Node) setupConn(newPeer *peer.Peer) {
	// alwayse suppose the remote site will build the connection in the same time;
	// compares the ID, the smaller one will be the client
	if crypto.PrivKeyToID(n.privKey) > newPeer.ID {
		time.Sleep(15 * time.Second)
	}
	n.connsMutex.Lock()
	_, ok := n.conns[newPeer.ID]
	n.connsMutex.Unlock()
	if ok {
		return
	}

	conn, err := utils.TCPConnectTo(newPeer.IP, newPeer.Port)
	if err != nil {
		logger.Warn("setup conection to %v failed:%v", newPeer, err)
		return
	}

	conn.SetSplitFunc(splitTCPStream)
	ec, err := n.ng.handshakeTo(conn, newPeer)
	if err != nil {
		logger.Warn("handshake to %v failed:%v", newPeer, err)
		conn.Disconnect()
		n.addNgBlackList(newPeer.ID)
		return
	}

	n.addConn(newPeer, conn, ec)
}

func (n *Node) recvHandshake(conn utils.TCPConn) {
	accept := false

	n.connsMutex.Lock()
	if len(n.conns) < n.maxPeersNum {
		accept = true
	}
	n.connsMutex.Unlock()

	peer, ec, err := n.ng.recvHandshake(conn, accept)
	if err != nil {
		logger.Warn("handle handshake from remote failed:%v\n", err)
		conn.Disconnect()
		return
	}

	if !accept {
		conn.Disconnect()
		return
	}

	n.connsMutex.Lock()
	// reject the duplicate connection with the same peer
	if _, ok := n.conns[peer.ID]; ok {
		accept = false
	}
	n.connsMutex.Unlock()

	if !accept {
		conn.Disconnect()
		return
	}

	n.addConn(peer, conn, ec)
}

func (n *Node) send(p Protocol, dp *PeerData) error {

	if len(n.conns) == 0 {
		return NoPeersError{}
	}

	// broadcast
	if len(dp.Peer) == 0 {
		n.connsMutex.Lock()
		for _, conn := range n.conns {
			conn.send(p.ID(), dp.Data)
		}
		n.connsMutex.Unlock()
		return nil
	}

	n.connsMutex.Lock()
	conn, ok := n.conns[dp.Peer]
	n.connsMutex.Unlock()
	if !ok {
		return PeerNotFoundError{Peer: dp.Peer}
	}

	conn.send(p.ID(), dp.Data)
	return nil
}

func (n *Node) addConn(peer *peer.Peer, conn utils.TCPConn, ec codec) {
	n.connsMutex.Lock()
	defer n.connsMutex.Unlock()

	if _, ok := n.conns[peer.ID]; !ok {
		c := newConn(peer, conn, ec, n.remoteRecv)
		n.conns[c.p.ID] = c
		conn.SetDisconnectCb(func(addr net.Addr) {
			logger.Debug("disconnect peer %v, address %v\n", peer.ID, addr)
			n.removeConn(peer.ID)
		})
		c.start()

		logger.Debug("add conn of %v\n", peer)
		return
	}

	logger.Debug("already exist a connection with %s\n", peer.ID)
	conn.Disconnect()
}

func (n *Node) removeConn(ID string) {
	n.delConnTask <- ID
}

func (n *Node) remoteRecv(peer string, protocolID uint8, data []byte) {
	// logger.Debug("recv a protocol[%d] packet, size %d\n", protocolID, len(data))
	if runner, ok := n.protocols[protocolID]; ok {
		select {
		case runner.Data <- &PeerData{
			Peer: peer,
			Data: data,
		}:
		default:
			logger.Warn("protocol %s recv packet queue full, drop it",
				runner.protocol.Name())
		}
	}
}

func (n *Node) addNgBlackList(peerID string) {
	n.ngMutex.Lock()
	defer n.ngMutex.Unlock()
	n.ngBlackList[peerID] = time.Now()
}

func (n *Node) cleanNgBlackList() {
	n.ngMutex.Lock()
	defer n.ngMutex.Unlock()

	curr := time.Now()
	for k, v := range n.ngBlackList {
		if curr.Sub(v) > 30*time.Minute {
			delete(n.ngBlackList, k)
		}
	}
}

func (n *Node) getExcludePeers() map[string]bool {
	result := make(map[string]bool)

	n.ngMutex.Lock()
	for k := range n.ngBlackList {
		result[k] = true
	}
	n.ngMutex.Unlock()

	n.connsMutex.Lock()
	for id := range n.conns {
		result[id] = true
	}
	n.connsMutex.Unlock()

	return result
}

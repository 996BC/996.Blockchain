package p2p

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/utils"
	"github.com/btcsuite/btcd/btcec"
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

// Node is a node that can communicate with others in the p2p network.
type Node interface {
	AddProtocol(p Protocol) ProtocolRunner
	Start()
	Stop()
}

// NewNode returns a p2p network Node
func NewNode(c *Config) Node {
	if c.Type != params.FullNode && c.Type != params.LightNode {
		logger.Fatal("invalid node type %d\n", c.Type)
	}
	n := &node{
		privKey:      c.PrivKey,
		chainID:      c.ChainID,
		nodeType:     c.Type,
		maxPeersNum:  c.MaxPeerNum,
		peerProvider: c.Provider,
		protocols:    make(map[uint8]*protocolRunner),
		ngBlackList:  make(map[string]time.Time),
		connectTask:  make(chan *peer.Peer, c.MaxPeerNum),
		connMgr:      newConnMgr(c.MaxPeerNum),
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

type node struct {
	tcpServer utils.TCPServer
	privKey   *btcec.PrivateKey
	chainID   uint8
	nodeType  params.NodeType

	maxPeersNum  int
	peerProvider peer.Provider

	protocolsMutex sync.Mutex
	protocols      map[uint8]*protocolRunner //<Protocol ID, ProtocolRunner>

	ng          *negotiator
	ngMutex     sync.Mutex
	ngBlackList map[string]time.Time

	connectTask chan *peer.Peer
	connMgr     connMgr

	lm *utils.LoopMode
}

// AddProtocol adds the runtime p2p network protocols
func (n *node) AddProtocol(p Protocol) ProtocolRunner {
	n.protocolsMutex.Lock()
	defer n.protocolsMutex.Unlock()

	if v, ok := n.protocols[p.ID()]; ok {
		logger.Fatal("protocol conflicts in ID:%s, exists:%s, wanted to add:%s",
			p.ID(), v.protocol.Name(), v.protocol.Name())
	}
	runner := newProtocolRunner(p, n.send)
	n.protocols[p.ID()] = runner
	return runner
}

func (n *node) Start() {
	if !n.tcpServer.Start() {
		logger.Fatalln("start node's tcp server failed")
	}
	n.connMgr.start()

	go n.loop()
	n.lm.StartWorking()
}

func (n *node) Stop() {
	if n.lm.Stop() {
		n.tcpServer.Stop()
		n.connMgr.stop()
	}
}

func (n *node) String() string {
	return fmt.Sprintf("[node] listen on %v", n.tcpServer.Addr())
}

func (n *node) loop() {
	n.lm.Add()
	defer n.lm.Done()

	getPeersToConnectTicker := time.NewTicker(10 * time.Second)
	statusReportTicker := time.NewTicker(15 * time.Second)
	ngBlackListCleanTicker := time.NewTicker(1 * time.Minute)

	acceptConn := n.tcpServer.GetTCPAcceptConnChannel()
	for {
		select {
		case <-n.lm.D:
			return
		case <-getPeersToConnectTicker.C:
			n.getPeersToConnect()
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

func (n *node) getPeersToConnect() {
	peersNum := n.connMgr.size()
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

func (n *node) statusReport() {
	if utils.GetLogLevel() < utils.LogDebugLevel {
		return
	}

	logger.Debug("current address book:%v\n", n.connMgr)
}

func (n *node) setupConn(newPeer *peer.Peer) {
	// alwayse suppose the remote site will build the connection in the same time;
	// compares the ID, the smaller one will be the client
	if crypto.PrivKeyToID(n.privKey) > newPeer.ID {
		time.Sleep(10 * time.Second)
	}
	if n.connMgr.isExist(newPeer.ID) {
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

func (n *node) recvHandshake(conn utils.TCPConn) {
	accept := false
	if n.connMgr.size() < n.maxPeersNum {
		accept = true
	}

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

	n.addConn(peer, conn, ec)
}

func (n *node) addConn(peer *peer.Peer, conn utils.TCPConn, ec codec) {
	if err := n.connMgr.add(peer, conn, ec, n.recv); err != nil {
		logger.Debug("addConn failed:%v\n", err)
		conn.Disconnect()
	}
}

func (n *node) send(p Protocol, dp *PeerData) error {
	return n.connMgr.send(p, dp)
}

func (n *node) recv(peer string, protocolID uint8, data []byte) {
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

func (n *node) addNgBlackList(peerID string) {
	n.ngMutex.Lock()
	defer n.ngMutex.Unlock()
	n.ngBlackList[peerID] = time.Now()
}

func (n *node) cleanNgBlackList() {
	n.ngMutex.Lock()
	defer n.ngMutex.Unlock()

	curr := time.Now()
	for k, v := range n.ngBlackList {
		if curr.Sub(v) > 30*time.Minute {
			delete(n.ngBlackList, k)
		}
	}
}

func (n *node) getExcludePeers() map[string]bool {
	result := make(map[string]bool)

	n.ngMutex.Lock()
	for k := range n.ngBlackList {
		result[k] = true
	}
	n.ngMutex.Unlock()

	connectedID := n.connMgr.getIDs()
	for _, id := range connectedID {
		result[id] = true
	}

	return result
}

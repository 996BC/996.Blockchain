package p2p

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/utils"
	"github.com/btcsuite/btcd/btcec"
)

var nodeTestVar = &struct {
	peers         []*peer.Peer
	connectionIDs []string
	ngBlackListID string

	// remotePeer will connect to our node
	remotePeer  *peer.Peer
	networkData []byte
}{
	peers:         []*peer.Peer{&peer.Peer{ID: "Peer_A"}, &peer.Peer{ID: "Peer_B"}},
	connectionIDs: []string{"Peer_C", "Peer_D"},
	ngBlackListID: "Peer_E",
	remotePeer:    &peer.Peer{ID: "Peer_F"},
	networkData:   []byte("hello world"),
}

func newNodeForTest() *node {
	tv := nodeTestVar

	key, _ := btcec.NewPrivateKey(btcec.S256())
	n := &node{
		privKey:      key,
		maxPeersNum:  128,
		peerProvider: newProviderMock(tv.peers),

		protocols: make(map[uint8]*protocolRunner),

		ng:          newNegotiatorMock(true),
		ngBlackList: make(map[string]time.Time),

		tcpConnectFunc: tcpConnectSuccMock,
		connectTask:    make(chan *peer.Peer, 128),
		connMgr:        newConnManagerMock(tv.connectionIDs),
	}
	n.ngBlackList[tv.ngBlackListID] = time.Now()

	return n
}

func TestGetPeersToConnect(t *testing.T) {
	tv := nodeTestVar
	n := newNodeForTest()
	provider := n.peerProvider.(*providerMock)
	n.getPeersToConnect()

	peers := make(map[string]bool)
	for {
		leave := false
		select {
		case peer := <-n.connectTask:
			peers[peer.ID] = false
		default:
			leave = true
		}
		if leave {
			break
		}
	}

	// verify
	// GetPeers() result
	for _, p := range tv.peers {
		if _, ok := peers[p.ID]; !ok {
			t.Fatalf("expect %s in get result\n", p.ID)
		}
	}

	// GetPeers() expectNum
	if err := utils.TCheckInt("GetPeers() expectNum", n.maxPeersNum-n.connMgr.size(), provider.getPeersExpect); err != nil {
		t.Fatal(err)
	}

	// GetPeers() exclude
	for _, id := range tv.connectionIDs {
		if _, ok := provider.getPeersExclude[id]; !ok {
			t.Fatalf("expect exclude connectionIDs %s\n", id)
		}
	}
	if _, ok := provider.getPeersExclude[tv.ngBlackListID]; !ok {
		t.Fatalf("exect exclude ngBlackListID %s\n", tv.ngBlackListID)
	}
}

func TestMaxPeerLimit(t *testing.T) {
	n := newNodeForTest()
	n.maxPeersNum = 1
	n.getPeersToConnect()

	select {
	case <-n.connectTask:
		t.Fatal("expect no connection tasks")
	default:
	}
}

func TestSetupConn(t *testing.T) {
	// success
	targetPeer := &peer.Peer{
		// it is biggest in lexicographical comparation
		ID: "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
	}
	n := newNodeForTest()
	n.setupConn(targetPeer)

	connManager := n.connMgr.(*connManagerMock)
	if err := utils.TCheckString("add peer ID", targetPeer.ID, connManager.addPeer.ID); err != nil {
		t.Fatal(err)
	}

	// fail via tcp connect
	n = newNodeForTest()
	n.tcpConnectFunc = tcpConnectFailMock
	n.setupConn(targetPeer)

	connManager = n.connMgr.(*connManagerMock)
	if connManager.addPeer != nil {
		t.Fatal("expect not adding peer")
	}

	// fail via negotiation
	n = newNodeForTest()
	n.ng = newNegotiatorMock(false)
	n.setupConn(targetPeer)

	connManager = n.connMgr.(*connManagerMock)
	if connManager.addPeer != nil {
		t.Fatal("expect not adding peer")
	}
}

func TestRecvConn(t *testing.T) {
	tv := nodeTestVar

	// success
	n := newNodeForTest()
	n.recvConn(newTCPConnMock())

	connManager := n.connMgr.(*connManagerMock)
	if err := utils.TCheckString("add peer ID", tv.remotePeer.ID, connManager.addPeer.ID); err != nil {
		t.Fatal(err)
	}

	// fail via maxPeer
	n = newNodeForTest()
	n.maxPeersNum = n.connMgr.size() // reach the limit, will not accept connection
	n.recvConn(newTCPConnMock())

	connManager = n.connMgr.(*connManagerMock)
	if connManager.addPeer != nil {
		t.Fatal("expect not adding peer")
	}

	// faild via handshake
	n = newNodeForTest()
	n.ng = newNegotiatorMock(false)
	n.recvConn(newTCPConnMock())

	connManager = n.connMgr.(*connManagerMock)
	if connManager.addPeer != nil {
		t.Fatal("expect not adding peer")
	}
}

func TestCleanNgBlackList(t *testing.T) {
	tv := nodeTestVar
	n := newNodeForTest()

	n.ngBlackList[tv.ngBlackListID] = time.Unix(0, 0)
	n.cleanNgBlackList()

	if err := utils.TCheckInt("node ngBlackList size", 0, len(n.ngBlackList)); err != nil {
		t.Fatal(err)
	}
}

func TestProtocolSend(t *testing.T) {
	tv := nodeTestVar
	n := newNodeForTest()
	p := &nodeTestProtocol{}

	runner := n.AddProtocol(p)
	runner.Send(&PeerData{
		Peer: tv.remotePeer.ID,
		Data: tv.networkData,
	})

	connManager := n.connMgr.(*connManagerMock)
	if err := utils.TCheckUint8("send protocol ID", p.ID(), connManager.sendProtocol.ID()); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckString("send peer", tv.remotePeer.ID, connManager.sendData.Peer); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("send data", tv.networkData, connManager.sendData.Data); err != nil {
		t.Fatal(err)
	}
}

func TestProtocolRecv(t *testing.T) {
	tv := nodeTestVar
	n := newNodeForTest()
	p := &nodeTestProtocol{}

	runner := n.AddProtocol(p)
	recvChan := runner.GetRecvChan()

	n.recv(tv.remotePeer.ID, p.ID(), tv.networkData)

	select {
	case pd := <-recvChan:
		if err := utils.TCheckString("recv peer", tv.remotePeer.ID, pd.Peer); err != nil {
			t.Fatal(err)
		}
		if err := utils.TCheckBytes("recv data", tv.networkData, pd.Data); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("expect recv data")
	}
}

///////////////////////////////////////providerMock
type providerMock struct {
	peers           []*peer.Peer
	getPeersExpect  int
	getPeersExclude map[string]bool
}

func newProviderMock(peers []*peer.Peer) *providerMock {
	return &providerMock{
		peers: peers,
	}
}

func (p *providerMock) Start() {}
func (p *providerMock) Stop()  {}
func (p *providerMock) GetPeers(expect int, exclude map[string]bool) ([]*peer.Peer, error) {
	p.getPeersExpect = expect
	p.getPeersExclude = exclude
	return p.peers, nil
}
func (p *providerMock) AddSeeds(seeds []*peer.Peer) {}

///////////////////////////////////////negotiatorMock
type negotiatorMock struct {
	success bool
}

func newNegotiatorMock(success bool) *negotiatorMock {
	return &negotiatorMock{
		success: success,
	}
}

func (n *negotiatorMock) handshakeTo(conn utils.TCPConn, peer *peer.Peer) (codec, error) {
	if n.success {
		return nil, nil
	}
	return nil, errors.New("")
}
func (n *negotiatorMock) recvHandshake(conn utils.TCPConn, accept bool) (*peer.Peer, codec, error) {
	tv := nodeTestVar
	if n.success {
		return tv.remotePeer, nil, nil
	}
	return nil, nil, errors.New("")
}

///////////////////////////////////////connManagerMock
type connManagerMock struct {
	ids     []string
	addPeer *peer.Peer

	sendProtocol Protocol
	sendData     *PeerData
}

func newConnManagerMock(ids []string) *connManagerMock {
	return &connManagerMock{
		ids: ids,
	}
}

func (c *connManagerMock) start() {}
func (c *connManagerMock) stop()  {}
func (c *connManagerMock) size() int {
	return len(c.ids)
}
func (c *connManagerMock) getIDs() []string {
	return c.ids
}
func (c *connManagerMock) isExist(peerID string) bool {
	for _, id := range c.ids {
		if id == peerID {
			return true
		}
	}

	return false
}
func (c *connManagerMock) send(p Protocol, dp *PeerData) error {
	c.sendProtocol = p
	c.sendData = dp
	return nil
}
func (c *connManagerMock) add(peer *peer.Peer, conn utils.TCPConn, ec codec, handler recvHandler) error {
	c.addPeer = peer
	return nil
}
func (c *connManagerMock) String() string {
	return ""
}

///////////////////////////////////////tcpConnectFuncMock
func tcpConnectSuccMock(ip net.IP, port int) (utils.TCPConn, error) {
	// tcpConnMock declare in negotiator_test.go
	return newTCPConnMock(), nil
}

func tcpConnectFailMock(ip net.IP, port int) (utils.TCPConn, error) {
	return nil, errors.New("")
}

///////////////////////////////////////nodeTestProtocol

type nodeTestProtocol struct {
}

func (n *nodeTestProtocol) ID() uint8 {
	return 64
}

func (n *nodeTestProtocol) Name() string {
	return "nodeTestProtocol"
}

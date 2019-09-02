package peer

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/serialize/discover"
	"github.com/996BC/996.Blockchain/utils"
)

var providerTestVar = &struct {
	p *provider

	// provider self infomation
	ip          net.IP
	port        int
	addr        *net.UDPAddr
	pubKeyHex   string
	pubKeyBytes []byte
	pubKey      *btcec.PublicKey

	// remote peer infomation
	remoteIP          net.IP
	remotePort        int
	remoteAddr        *net.UDPAddr
	remotePubKeyHex   string
	remotePubKeyBytes []byte
	remotePubKey      *btcec.PublicKey
}{
	ip:        net.ParseIP("192.168.1.1"),
	port:      10000,
	pubKeyHex: "029ab6627bffd4ee5b5a6f1f96b2730980ca14033bd6f3a63764c1f1aedd3634eb",

	remoteIP:        net.ParseIP("192.168.1.2"),
	remotePort:      10081,
	remotePubKeyHex: "031ceff33b51b2b013cc7139981dad78f919bdf4d9d0f767cfdec6ed96c8b2492b",
}

func init() {
	tv := providerTestVar

	tv.addr = &net.UDPAddr{IP: tv.ip, Port: tv.port}
	tv.pubKeyBytes, _ = utils.FromHex(tv.pubKeyHex)
	tv.pubKey, _ = btcec.ParsePubKey(tv.pubKeyBytes, btcec.S256())

	tv.remoteAddr = &net.UDPAddr{IP: tv.remoteIP, Port: tv.remotePort}
	tv.remotePubKeyBytes, _ = utils.FromHex(tv.remotePubKeyHex)
	tv.remotePubKey, _ = btcec.ParsePubKey(tv.remotePubKeyBytes, btcec.S256())

	tv.p = &provider{
		ip:            tv.ip,
		port:          tv.port,
		compressedKey: tv.pubKeyBytes,
		udp:           newUDPServerMock(),
		table:         newTableStub(),
		pingHash:      make(map[string]time.Time),
	}
}

func TestPing(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	p.ping()

	// verify reqeust
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(1); err != nil {
		t.Fatal(err)
	}
	reqPkt, _ := udpMock.pop()

	if err := utils.TCheckAddr("request address", tv.remoteAddr, reqPkt.Addr); err != nil {
		t.Fatal(err)
	}

	pingPkt := reqPkt.Data
	ping, err := discover.UnmarshalPing(bytes.NewReader(pingPkt))
	if err != nil {
		t.Fatal("unmarshal Ping failed\n")
	}
	if err := utils.TCheckBytes("request public key", tv.pubKeyBytes, ping.PubKey); err != nil {
		t.Fatal(err)
	}

	// check pingHash and cleanup
	pingHashKey := utils.ToHex(utils.Hash(pingPkt))
	if _, ok := p.pingHash[pingHashKey]; !ok {
		t.Fatalf("expect existing pingHash %s\n", pingHashKey)
	}
	delete(p.pingHash, pingHashKey)
}

func TestGetNeighbours(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	p.getNeighbours()

	// verify reqeust
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(1); err != nil {
		t.Fatal(err)
	}
	reqPkt, _ := udpMock.pop()

	if err := utils.TCheckAddr("request address", tv.remoteAddr, reqPkt.Addr); err != nil {
		t.Fatal(err)
	}

	getNeighbourPkt := reqPkt.Data
	getNeighbour, err := discover.UnmarshalGetNeighbours(bytes.NewReader(getNeighbourPkt))
	if err != nil {
		t.Fatal("unmarshal GetNeighbour failed\n")
	}
	if err := utils.TCheckBytes("request public key", tv.pubKeyBytes, getNeighbour.PubKey); err != nil {
		t.Fatal(err)
	}
}

func TestHandlePing(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	remotePingPkt := discover.NewPing(tv.remotePubKeyBytes).Marshal()
	p.handlePing(remotePingPkt, tv.remoteAddr)

	// verify response
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(1); err != nil {
		t.Fatal(err)
	}
	respPkt, _ := udpMock.pop()

	if err := utils.TCheckAddr("response address", tv.remoteAddr, respPkt.Addr); err != nil {
		t.Fatal(err)
	}

	pongPkt := respPkt.Data
	pong, err := discover.UnmarshalPong(bytes.NewReader(pongPkt))
	if err != nil {
		t.Fatal("unmarshal pong failed\n")
	}
	if err := utils.TCheckBytes("ping hash", pong.PingHash, utils.Hash(remotePingPkt)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("response public key", p.compressedKey, pong.PubKey); err != nil {
		t.Fatal(err)
	}
}

func TestHandlePong(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	pingHash := utils.Hash([]byte("a_ping_hash"))
	pingHashKey := utils.ToHex(pingHash)
	pongPkt := discover.NewPong(pingHash, tv.remotePubKeyBytes).Marshal()

	p.pingHash[pingHashKey] = time.Now()
	p.handlePong(pongPkt, tv.remoteAddr)

	if len(p.pingHash) != 0 {
		t.Fatal("expect clean pingHash after handle pong\n")
	}
}

func TestHandleGetNeighboursNotFromMyPeers(t *testing.T) {
	p := providerTestVar.p

	unknownPeerKeyHex := "0429a89e5077cb36383e81b43bd06e4aa0bc7ffd55acf15e5406233614b5f95942a33e817823a04d606c6309cbfdee533340a679d734883beea5101b69cb309c4e"
	unknownPeerKeyBytes, _ := utils.FromHex(unknownPeerKeyHex)

	getNeighboursPkt := discover.NewGetNeighbours(unknownPeerKeyBytes).Marshal()
	p.handleGetNeigoubours(getNeighboursPkt, &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 999})

	// verify response
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(0); err != nil {
		t.Fatal(err)
	}
}

func TestHandleGetNeighbours(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	getNeighboursPkt := discover.NewGetNeighbours(tv.remotePubKeyBytes).Marshal()
	p.handleGetNeigoubours(getNeighboursPkt, tv.remoteAddr)

	// verify response
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(2); err != nil {
		t.Fatal(err)
	}
	pkt, _ := udpMock.pop()

	if err := utils.TCheckAddr("reponse address", tv.remoteAddr, pkt.Addr); err != nil {
		t.Fatal(err)
	}

	neighbours, err := discover.UnmarshalNeighbours(bytes.NewReader(pkt.Data))
	if err != nil {
		t.Fatal("unmarshal Neighbours failed\n")
	}

	if len(neighbours.Nodes) != 1 {
		t.Fatal("expect 1 node in Neighbours\n")
	}

	node := neighbours.Nodes[0]
	if err := utils.TCheckBytes("node public key", tv.remotePubKeyBytes, node.PubKey); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckIP("node IP", tv.remoteIP, node.Addr.IP); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("node port", tv.remotePort, int(node.Addr.Port)); err != nil {
		t.Fatal(err)
	}
}

func TestHandleNeighbours(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	neighbourPkt := discover.NewNeighbours([]*discover.Node{
		discover.NewNode(discover.NewAddress(tv.remoteIP.String(), int32(tv.remotePort)), tv.remotePubKeyBytes),
	}).Marshal()
	p.handleNeigoubours(neighbourPkt, tv.remoteAddr)

	// verify add result
	table := p.table.(*tableMock)
	if err := utils.TCheckInt("table add list size", 1, len(table.add)); err != nil {
		t.Fatal(err)
	}

	node := table.add[0]
	if err := utils.TCheckIP("node IP", tv.remoteIP, node.IP); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("node port", tv.remotePort, node.Port); err != nil {
		t.Fatal(err)
	}
}

/////////////////////////////////////////////////tableMock

type tableMock struct {
	peer *Peer
	add  []*Peer
}

func newTableStub() *tableMock {
	return &tableMock{
		peer: NewPeer(providerTestVar.remoteIP, providerTestVar.remotePort, providerTestVar.remotePubKey),
	}
}
func (t *tableMock) addPeers(p []*Peer, isSeed bool) {
	t.add = p
}
func (t *tableMock) getPeers(expect int, exclude map[string]bool) []*Peer {
	return []*Peer{t.peer}
}
func (t *tableMock) exists(id string) bool {
	return id == t.peer.ID
}
func (t *tableMock) getPeersToPing() []*Peer {
	return []*Peer{t.peer}
}
func (t *tableMock) getPeersToGetNeighbours() []*Peer {
	return []*Peer{t.peer}
}
func (t *tableMock) recvPing(p *Peer) {}
func (t *tableMock) recvPong(p *Peer) {}
func (t *tableMock) refresh()         {}

////////////////////////////////////////////////udpServerMock

type udpServerMock struct {
	sendQ []*utils.UDPPacket
}

func newUDPServerMock() *udpServerMock {
	return &udpServerMock{}
}
func (u *udpServerMock) GetRecvChannel() <-chan *utils.UDPPacket {
	return nil
}
func (u *udpServerMock) Send(packet *utils.UDPPacket) {
	u.sendQ = append(u.sendQ, packet)
}
func (u *udpServerMock) Start() bool {
	return true
}
func (u *udpServerMock) Stop() {}
func (u *udpServerMock) checkSendQSize(expect int) error {
	return utils.TCheckInt("udp send queue size", expect, len(u.sendQ))
}
func (u *udpServerMock) pop() (*utils.UDPPacket, error) {
	if len(u.sendQ) == 0 {
		return nil, fmt.Errorf("empty sendQ")
	}

	result := u.sendQ[0]
	u.sendQ = u.sendQ[1:]
	return result, nil
}

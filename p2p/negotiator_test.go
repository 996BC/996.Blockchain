package p2p

import (
	"bytes"
	"net"
	"testing"

	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/serialize/handshake"
	"github.com/996BC/996.Blockchain/utils"
	"github.com/btcsuite/btcd/btcec"
)

var negotiatorTestVar = &struct {
	sendPrivKey        *btcec.PrivateKey
	sendPubKey         *btcec.PublicKey
	sendSessionPrivKey *btcec.PrivateKey
	sendSessionPubKey  *btcec.PublicKey

	recvPrivKey        *btcec.PrivateKey
	recvPubKey         *btcec.PublicKey
	recvSessionPrivKey *btcec.PrivateKey
	recvSessionPubKey  *btcec.PublicKey

	expectCodec codec

	remoteIP   net.IP
	remotePort int

	chainID    uint8
	errChainID uint8
}{}

func init() {
	tv := negotiatorTestVar

	sendPrivKeyHex := "952B8927FEE7B348A8C8B7164EF550F89DDE3FCCFC177C0E41B033325296EF63"
	keyBytes, _ := utils.FromHex(sendPrivKeyHex)
	tv.sendPrivKey, tv.sendPubKey = btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

	sendSessionPrivKeyHex := "BF40216703E409988F9E07EFFF851AE8AD53B2EC9193D1CE7CB28AB066274466"
	keyBytes, _ = utils.FromHex(sendSessionPrivKeyHex)
	tv.sendSessionPrivKey, tv.sendSessionPubKey = btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

	recvPrivKeyHex := "2203355A167D13F7BFD751777B9C68655F1466526E7594876325CF659203BAC6"
	keyBytes, _ = utils.FromHex(recvPrivKeyHex)
	tv.recvPrivKey, tv.recvPubKey = btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

	recvSessionPrivKeyHex := "B9F7952D389470A15E996642DDCD099C9C557F8444D730BA79A3E56BCDF671CA"
	keyBytes, _ = utils.FromHex(recvSessionPrivKeyHex)
	tv.recvSessionPrivKey, tv.recvSessionPubKey = btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)

	tv.expectCodec, _ = newAESGCMCodec(tv.recvSessionPubKey, tv.sendSessionPrivKey)

	tv.remoteIP = net.ParseIP("192.168.1.2")
	tv.remotePort = 10000

	tv.chainID = 1
	tv.errChainID = 2
}

func TestHandshakeTo(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	receiver := newReceiver(params.FullNode)
	conn := newTCPConnMock()

	// mock accept response
	resp := receiver.genAcceptResponse(tv.recvSessionPrivKey)
	conn.setRecvPkt(resp)

	// a full node handshake to another full node
	peer := peer.NewPeer(tv.remoteIP, tv.remotePort, tv.recvPubKey)
	codec, err := sender.handshakeTo(conn, peer)
	if err != nil {
		t.Fatalf("handshakeTo err:%v\n", err)
	}

	// check handshake request
	_, reqPkt, _ := verifyTCPPacket(conn.getSendPkt())
	req, err := handshake.UnmarshalRequest(bytes.NewBuffer(reqPkt))
	if err != nil {
		t.Fatalf("unmashal request failed:%v\n", err)
	}
	checkRequest(t, req)

	// check codec
	checkCodec(t, codec, tv.expectCodec)
}

func TestRecvHandshake(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	receiver := newReceiver(params.FullNode)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey, tv.recvPubKey)
	conn.setRecvPkt(req)

	// a full node wait another full node handshake
	peer, codec, err := receiver.recvHandshake(conn, true)
	if err != nil {
		t.Fatalf("recvHandshake err:%v\n", err)
	}

	// check handshake response
	_, respPkt, _ := verifyTCPPacket(conn.getSendPkt())
	resp, err := handshake.UnmarshalResponse(bytes.NewReader(respPkt))
	if err != nil {
		t.Fatalf("unmarshal response failed:%v\n", err)
	}
	checkResponse(t, resp)

	// check peer
	checkPeer(t, peer)

	// check codec
	checkCodec(t, codec, tv.expectCodec)
}

func TestReject(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	receiver := newReceiver(params.FullNode)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey, tv.recvPubKey)
	conn.setRecvPkt(req)

	// reject request
	_, _, err := receiver.recvHandshake(conn, false)
	if err != nil {
		t.Fatalf("decrypt response failed:%v\n", err)
	}

	// check handshake response
	_, respPkt, _ := verifyTCPPacket(conn.getSendPkt())
	resp, err := handshake.UnmarshalResponse(bytes.NewReader(respPkt))
	if err != nil {
		t.Fatalf("unmarshal response failed:%v\n", err)
	}

	err = sender.whetherRejectResp(resp, tv.recvPubKey)
	if _, ok := err.(NegotiateGotRejection); !ok {
		t.Fatal("expect get rejection")
	}
}

func TestChainIDMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	sender.chainID = tv.errChainID
	receiver := newReceiver(params.FullNode)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey, tv.recvPubKey)
	conn.setRecvPkt(req)

	_, _, err := receiver.recvHandshake(conn, true)
	if _, ok := err.(NegotiateChainIDMismatch); !ok {
		t.Fatalf("expect chain ID mismatch error, %v\n", err)
	}
}

func TestSenderNodeTypeMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.LightNode)
	receiver := newReceiver(params.LightNode)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey, tv.recvPubKey)
	conn.setRecvPkt(req)

	_, _, err := receiver.recvHandshake(conn, true)
	if _, ok := err.(NegotiateNodeTypeMismatch); !ok {
		t.Fatalf("expect node type mismatch error, %v\n", err)
	}
}

func TestReceiverNodeTypeMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	receiver := newReceiver(params.LightNode)
	conn := newTCPConnMock()

	// mock accept response
	resp := receiver.genAcceptResponse(tv.recvSessionPrivKey)
	conn.setRecvPkt(resp)

	peer := peer.NewPeer(tv.remoteIP, tv.remotePort, tv.recvPubKey)
	_, err := sender.handshakeTo(conn, peer)
	if _, ok := err.(NegotiateNodeTypeMismatch); !ok {
		t.Fatalf("expect node type mismatch error, %v\n", err)
	}
}

func TestSenderCodeVersionMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	receiver := newReceiver(params.FullNode)
	receiver.minimizeVersionRequired++
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey, tv.recvPubKey)
	conn.setRecvPkt(req)

	_, _, err := receiver.recvHandshake(conn, true)
	if _, ok := err.(NegotiateCodeVersionMismatch); !ok {
		t.Fatalf("expect code version mismatch error, %v\n", err)
	}
}

func TestReceiverCoderVersionMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(params.FullNode)
	sender.minimizeVersionRequired++
	receiver := newReceiver(params.LightNode)
	conn := newTCPConnMock()

	// mock accept response
	resp := receiver.genAcceptResponse(tv.recvSessionPrivKey)
	conn.setRecvPkt(resp)

	peer := peer.NewPeer(tv.remoteIP, tv.remotePort, tv.recvPubKey)
	_, err := sender.handshakeTo(conn, peer)
	if _, ok := err.(NegotiateCodeVersionMismatch); !ok {
		t.Fatalf("expect code version mismatch error, %v\n", err)
	}
}

func newSender(nodeType params.NodeType) *negotiatorImp {
	tv := negotiatorTestVar
	ng := newNegotiator(tv.sendPrivKey, tv.chainID, nodeType)
	result := ng.(*negotiatorImp)
	result.genSessionKeyFunc = senderGenSessionKeyFunc
	return result
}

func newReceiver(nodeType params.NodeType) *negotiatorImp {
	tv := negotiatorTestVar
	ng := newNegotiator(tv.recvPrivKey, tv.chainID, nodeType)
	result := ng.(*negotiatorImp)
	result.genSessionKeyFunc = receiverGenSessionKeyFunc
	return result
}

func checkRequest(t *testing.T, req *handshake.Request) {
	tv := negotiatorTestVar

	if !req.Verify() {
		t.Fatal("verify request failed")
	}
	if err := utils.TCheckUint8("version", handshake.HandshakeV1, req.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("chain id", tv.chainID, req.ChainID); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint16("code version", uint16(params.CurrentCodeVersion),
		uint16(req.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node type", uint8(params.FullNode), uint8(req.NodeType)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("long term key", tv.sendPubKey.SerializeCompressed(), req.PubKey); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", tv.sendSessionPubKey.SerializeCompressed(), req.SessionKey); err != nil {
		t.Fatal(err)
	}
}

func checkResponse(t *testing.T, resp *handshake.Response) {
	tv := negotiatorTestVar

	if !resp.Verify(tv.recvPubKey) {
		t.Fatal("verify response failed")
	}
	if err := utils.TCheckUint8("version", handshake.HandshakeV1, resp.Version); err != nil {
		t.Fatal(err)
	}
	if !resp.IsAccept() {
		t.Fatal("expect accept")
	}
	if err := utils.TCheckUint16("code version", uint16(params.CurrentCodeVersion),
		uint16(resp.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node type", uint8(params.FullNode), uint8(resp.NodeType)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", tv.recvSessionPubKey.SerializeCompressed(), resp.SessionKey); err != nil {
		t.Fatal(err)
	}
}

func checkPeer(t *testing.T, p *peer.Peer) {
	tv := negotiatorTestVar

	if err := utils.TCheckIP("peer IP", tv.remoteIP, p.IP); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("peer port", tv.remotePort, p.Port); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("peer key", tv.sendPubKey.SerializeCompressed(), p.Key.SerializeCompressed()); err != nil {
		t.Fatal(err)
	}
}

func checkCodec(t *testing.T, result codec, expect codec) {
	originText := []byte("nieogitator test codec check")

	// result encrypt, expect decrypt
	cipherText, err := result.encrypt(originText)
	if err != nil {
		t.Fatalf("result codec encrypt failed:%v\n", err)
	}
	plainText, err := expect.decrypt(cipherText)
	if err != nil {
		t.Fatalf("expect codec decrypt failed:%v\n", err)
	}

	if err := utils.TCheckBytes("plain text", originText, plainText); err != nil {
		t.Fatal(err)
	}

	// expect encrypt, result decrypt
	cipherText, _ = expect.encrypt(originText)
	if plainText, err = result.decrypt(cipherText); err != nil {
		t.Fatalf("result codec decrypt failed:%v\n", err)
	}
	if err := utils.TCheckBytes("plain text", originText, plainText); err != nil {
		t.Fatal(err)
	}
}

///////////////////////////////////////genSessionKeyFuncStub

func senderGenSessionKeyFunc() (*btcec.PrivateKey, error) {
	tv := negotiatorTestVar
	return tv.sendSessionPrivKey, nil
}

func receiverGenSessionKeyFunc() (*btcec.PrivateKey, error) {
	tv := negotiatorTestVar
	return tv.recvSessionPrivKey, nil
}

///////////////////////////////////////tcpConnMock

type tcpConnMock struct {
	sendPkt []byte
	recvQ   chan []byte
}

func newTCPConnMock() *tcpConnMock {
	return &tcpConnMock{
		recvQ: make(chan []byte, 128),
	}
}

func (t *tcpConnMock) Send(data []byte) {
	t.sendPkt = data
}
func (t *tcpConnMock) GetRecvChannel() <-chan []byte {
	return t.recvQ
}
func (t *tcpConnMock) SetSplitFunc(func(received *bytes.Buffer) ([][]byte, error)) {}
func (t *tcpConnMock) SetDisconnectCb(func(addr net.Addr))                         {}
func (t *tcpConnMock) RemoteAddr() net.Addr {
	tv := negotiatorTestVar
	return &net.TCPAddr{
		IP:   tv.remoteIP,
		Port: tv.remotePort,
	}
}
func (t *tcpConnMock) Disconnect() {}
func (t *tcpConnMock) getSendPkt() []byte {
	return t.sendPkt
}
func (t *tcpConnMock) setRecvPkt(data []byte) {
	t.recvQ <- data
}

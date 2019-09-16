package p2p

import (
	"bytes"
	"fmt"
	"time"

	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/serialize/handshake"
	"github.com/996BC/996.Blockchain/utils"
	"github.com/btcsuite/btcd/btcec"
)

/*
sender:
	1. generate random session used temporary key
	2. use self long-term key to sign message
	3. send
receiver:
	1. generate random session used temporary key
	3. use self long-term key to sign message
	5. reply
final:
	1. get shared secret P from two temporary key
	2. sha512(P), use first 32 bytes as secret key and rest 12 bytes as nonce
	3. use AES-GCM-256 to encrypt/decrypt following message
*/

const (
	handshakeProtocolID = 0
	nonceSize           = 12
)

type negotiator interface {
	handshakeTo(conn utils.TCPConn, peer *peer.Peer) (codec, error)
	recvHandshake(conn utils.TCPConn, accept bool) (*peer.Peer, codec, error)
}

type negotiatorImp struct {
	privKey                 *btcec.PrivateKey
	pubKey                  *btcec.PublicKey
	chainID                 uint8
	nodeType                params.NodeType
	codeVersion             params.CodeVersion
	minimizeVersionRequired params.CodeVersion
	genSessionKeyFunc       func() (*btcec.PrivateKey, error) // for test stub
}

func newNegotiator(privKey *btcec.PrivateKey, chainID uint8, nodeType params.NodeType) negotiator {
	result := &negotiatorImp{
		privKey:                 privKey,
		chainID:                 chainID,
		nodeType:                nodeType,
		codeVersion:             params.CurrentCodeVersion,
		minimizeVersionRequired: params.MinimizeVersionRequired,
		genSessionKeyFunc:       genSessionKeyFunc,
	}
	result.pubKey = privKey.PubKey()
	return result
}

func (n *negotiatorImp) handshakeTo(conn utils.TCPConn, peer *peer.Peer) (codec, error) {
	// session temporary key, temporary nonce
	sessionPrivKey, err := n.genSessionKeyFunc()
	if err != nil {
		return nil, err
	}

	// send handshake request
	requestBytes := n.genRequest(sessionPrivKey, peer.Key)
	conn.Send(requestBytes)

	// wait handshake response
	response, err := n.waitResponse(conn, sessionPrivKey)
	if err != nil {
		return nil, err
	}

	if err := n.whetherRejectResp(response, peer.Key); err != nil {
		return nil, err
	}

	peerSessionKey, err := btcec.ParsePubKey(response.SessionKey, btcec.S256())
	if err != nil {
		return nil, err
	}

	return newAESGCMCodec(peerSessionKey, sessionPrivKey)
}

func (n *negotiatorImp) recvHandshake(conn utils.TCPConn, accept bool) (*peer.Peer, codec, error) {
	request, err := n.waitRequest(conn)
	if err != nil {
		return nil, nil, err
	}

	if !request.Verify() {
		return nil, nil, NegotiateVerifySigFailed{}
	}

	peerSessionKey, err := btcec.ParsePubKey(request.SessionKey, btcec.S256())
	if err != nil {
		return nil, nil, NegotiateBrokenData{
			info: fmt.Sprintf("parse handshake session public key failed:%v", err),
		}
	}

	// reject
	if !accept {
		rejectRsp := n.genRejectResponse()
		conn.Send(rejectRsp)
		return nil, nil, nil
	}

	if err := n.whetherRejectReq(request); err != nil {
		return nil, nil, err
	}

	// accept
	// session temporary key, temporary nonce
	sessionPrivKey, err := n.genSessionKeyFunc()
	if err != nil {
		return nil, nil, err
	}

	acceptRsp := n.genAcceptResponse(sessionPrivKey)
	conn.Send(acceptRsp)

	ec, err := newAESGCMCodec(peerSessionKey, sessionPrivKey)
	if err != nil {
		return nil, nil, err
	}

	peer, err := n.getPeerFromRequest(conn, request)
	if err != nil {
		return nil, nil, err
	}

	return peer, ec, nil
}

func (n *negotiatorImp) waitResponse(conn utils.TCPConn, sessionPrivKey *btcec.PrivateKey) (*handshake.Response, error) {
	plainText, err := n.readPacket(conn)
	if err != nil {
		return nil, err
	}

	resp, err := handshake.UnmarshalResponse(bytes.NewReader(plainText))
	if err != nil {
		return nil, NegotiateBrokenData{
			info: fmt.Sprintf("unmarshal handshake response failed:%v", err),
		}
	}
	return resp, nil
}

func (n *negotiatorImp) waitRequest(conn utils.TCPConn) (*handshake.Request, error) {
	plainText, err := n.readPacket(conn)
	if err != nil {
		return nil, err
	}

	request, err := handshake.UnmarshalRequest(bytes.NewReader(plainText))
	if err != nil {
		return nil, NegotiateBrokenData{
			info: fmt.Sprintf("unmarshal handshake request failed:%v", err),
		}
	}

	return request, nil
}

func (n *negotiatorImp) genRequest(sessionPrivKey *btcec.PrivateKey,
	peerKey *btcec.PublicKey) []byte {

	sessionPubKey := sessionPrivKey.PubKey()
	sessionPubKeyBytes := sessionPubKey.SerializeCompressed()

	req := handshake.NewRequestV1(n.chainID, n.codeVersion, n.nodeType,
		n.pubKey.SerializeCompressed(), sessionPubKeyBytes)
	req.Sign(n.privKey)

	return buildTCPPacket(req.Marshal(), handshakeProtocolID)
}

func (n *negotiatorImp) genRejectResponse() []byte {
	resp := handshake.NewRejectResponseV1()
	resp.Sign(n.privKey)

	return buildTCPPacket(resp.Marshal(), handshakeProtocolID)
}

func (n *negotiatorImp) genAcceptResponse(sessionPrivKey *btcec.PrivateKey) []byte {
	resp := handshake.NewAcceptResponseV1(n.codeVersion, n.nodeType,
		sessionPrivKey.PubKey().SerializeCompressed())
	resp.Sign(n.privKey)

	return buildTCPPacket(resp.Marshal(), handshakeProtocolID)
}

func (n *negotiatorImp) readPacket(conn utils.TCPConn) ([]byte, error) {
	timeoutTicker := time.NewTicker(5 * time.Second)
	recvC := conn.GetRecvChannel()
	var payload []byte
	var protocolID uint8
	var ok bool

	select {
	case <-timeoutTicker.C:
		return nil, NegotiateTimeout{}
	case packet := <-recvC:
		if ok, payload, protocolID = verifyTCPPacket(packet); !ok {
			return nil, NegotiateBrokenData{
				info: fmt.Sprintf("veirfy handshake packet checksum failed"),
			}
		}
	}

	if protocolID != handshakeProtocolID {
		return nil, NegotiateBrokenData{
			info: fmt.Sprintf("invalid protocol ID for handshake %d", protocolID),
		}
	}

	return payload, nil
}

func (n *negotiatorImp) whetherRejectReq(request *handshake.Request) error {
	if request.ChainID != n.chainID {
		return NegotiateChainIDMismatch{}
	}

	if request.CodeVersion < n.minimizeVersionRequired {
		return NegotiateCodeVersionMismatch{n.minimizeVersionRequired, request.CodeVersion}
	}

	if n.nodeType == params.LightNode && request.NodeType == params.LightNode {
		return NegotiateNodeTypeMismatch{}
	}

	return nil
}

func (n *negotiatorImp) whetherRejectResp(response *handshake.Response, remotePubKey *btcec.PublicKey) error {
	if !response.Verify(remotePubKey) {
		return NegotiateVerifySigFailed{}
	}

	if !response.IsAccept() {
		return NegotiateGotRejection{}
	}

	if response.CodeVersion < n.minimizeVersionRequired {
		return NegotiateCodeVersionMismatch{n.minimizeVersionRequired, response.CodeVersion}
	}

	if response.NodeType == params.LightNode {
		return NegotiateNodeTypeMismatch{}
	}

	return nil
}

func (n *negotiatorImp) getPeerFromRequest(conn utils.TCPConn, request *handshake.Request) (*peer.Peer, error) {
	peerPubKey, err := btcec.ParsePubKey(request.PubKey, btcec.S256())
	if err != nil {
		return nil, NegotiateBrokenData{
			info: fmt.Sprintf("parse handhshake peer public key failed:%v", err),
		}
	}
	addr := conn.RemoteAddr()
	ip, port := utils.ParseIPPort(addr.String())
	peer := peer.NewPeer(ip, port, peerPubKey)
	return peer, nil
}

func genSessionKeyFunc() (*btcec.PrivateKey, error) {
	sessionPrivKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}

	return sessionPrivKey, nil
}

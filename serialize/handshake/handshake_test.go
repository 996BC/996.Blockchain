package handshake

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/utils"
)

func TestRequest(t *testing.T) {
	longtermKey, _ := btcec.NewPrivateKey(btcec.S256())
	longtermPubKeyBytes := longtermKey.PubKey().SerializeCompressed()
	sessionPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	chainID := uint8(1)
	codeVersion := params.CodeVersion(1)
	nodeType := params.FullNode
	sessionKey := sessionPrivKey.PubKey().SerializeCompressed()

	request := NewRequestV1(chainID, codeVersion, nodeType, longtermPubKeyBytes, sessionKey)
	request.Sign(longtermKey)
	requestBytes := request.Marshal()

	rRequest, err := UnmarshalRequest(bytes.NewReader(requestBytes))
	if err != nil {
		t.Fatalf("unmarshal Request failed:%v\n", err)
	}

	if err := utils.TCheckUint8("version", HandshakeV1, rRequest.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("chain ID", chainID, rRequest.ChainID); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint16("code version", uint16(codeVersion), uint16(rRequest.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node type", nodeType, rRequest.NodeType); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("public key", longtermPubKeyBytes, rRequest.PubKey); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", sessionKey, rRequest.SessionKey); err != nil {
		t.Fatal(err)
	}
	if !rRequest.Verify() {
		t.Fatal("verify failed\n")
	}
}

func TestAcceptResponse(t *testing.T) {
	longtermPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	longtermPubKey := longtermPrivKey.PubKey()

	sessionPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	codeVersion := params.CodeVersion(2)
	nodeType := params.LightNode
	sessionKey := sessionPrivKey.PubKey().SerializeCompressed()

	acceptResponce := NewAcceptResponseV1(codeVersion, nodeType, sessionKey)
	acceptResponce.Sign(longtermPrivKey)
	acceptResponceBytes := acceptResponce.Marshal()

	rAcceptResponse, err := UnmarshalResponse(bytes.NewReader(acceptResponceBytes))
	if err != nil {
		t.Fatalf("unmarshal Response failed:%v\n", err)
	}

	if err := utils.TCheckUint8("version", HandshakeV1, rAcceptResponse.Version); err != nil {
		t.Fatal(err)
	}
	if !rAcceptResponse.IsAccept() {
		t.Fatal("expect accept\n")
	}
	if err := utils.TCheckUint16("code version", uint16(codeVersion), uint16(rAcceptResponse.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node type", nodeType, rAcceptResponse.NodeType); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", sessionKey, rAcceptResponse.SessionKey); err != nil {
		t.Fatal(err)
	}
	if !rAcceptResponse.Verify(longtermPubKey) {
		t.Fatal("verify failed\n")
	}
}

func TestRejectResponse(t *testing.T) {
	longtermPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	longtermPubKey := longtermPrivKey.PubKey()

	rejectResponse := NewRejectResponseV1()
	rejectResponse.Sign(longtermPrivKey)
	rejectResponseBytes := rejectResponse.Marshal()

	rRejectResponse, err := UnmarshalResponse(bytes.NewReader(rejectResponseBytes))
	if err != nil {
		t.Fatalf("unmarshal Response failed:%v\n", err)
	}
	if rRejectResponse.IsAccept() {
		t.Fatal("expect reject")
	}
	if !rRejectResponse.Verify(longtermPubKey) {
		t.Fatal("verify failed\n")
	}
}

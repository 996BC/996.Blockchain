package handshake

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	acceptFlag = uint8(1)
	rejectFlag = uint8(2)
)

type Response struct {
	Version     uint8
	Accept      uint8
	CodeVersion params.CodeVersion
	NodeType    params.NodeType
	SessionKey  []byte
	Sig         []byte
}

func NewAcceptResponseV1(codeVersion params.CodeVersion, nodeType params.NodeType,
	sessionKey []byte) *Response {
	return &Response{
		Version:     HandshakeV1,
		Accept:      acceptFlag,
		CodeVersion: codeVersion,
		NodeType:    nodeType,
		SessionKey:  sessionKey,
	}
}

func NewRejectResponseV1() *Response {
	return &Response{
		Version: HandshakeV1,
		Accept:  rejectFlag,
	}
}

func UnmarshalResponse(data io.Reader) (*Response, error) {
	result := &Response{}
	var sessionKeyLen uint8
	var sigLen uint16
	var err error

	if err = binary.Read(data, binary.BigEndian, &result.Version); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.Accept); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.CodeVersion); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.NodeType); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &sessionKeyLen); err != nil {
		return nil, err
	}
	result.SessionKey = make([]byte, sessionKeyLen)
	if err = binary.Read(data, binary.BigEndian, result.SessionKey); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &sigLen); err != nil {
		return nil, err
	}
	result.Sig = make([]byte, sigLen)
	if err = binary.Read(data, binary.BigEndian, result.Sig); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Response) Marshal() []byte {
	result := new(bytes.Buffer)

	binary.Write(result, binary.BigEndian, r.Version)
	binary.Write(result, binary.BigEndian, r.Accept)
	binary.Write(result, binary.BigEndian, r.CodeVersion)
	binary.Write(result, binary.BigEndian, r.NodeType)

	sessionKeyLen := utils.Uint8Len(r.SessionKey)
	binary.Write(result, binary.BigEndian, sessionKeyLen)
	binary.Write(result, binary.BigEndian, r.SessionKey)

	sigLen := utils.Uint16Len(r.Sig)
	binary.Write(result, binary.BigEndian, sigLen)
	binary.Write(result, binary.BigEndian, r.Sig)

	return result.Bytes()
}

// Sign generate the signature for HandshakeResponse and set to the field Sig
func (r *Response) Sign(privKey *btcec.PrivateKey) {
	sig, _ := privKey.Sign(r.getSignContentHash())
	r.Sig = sig.Serialize()
}

// Verify checks the response is valid or not
func (r *Response) Verify(pubKey *btcec.PublicKey) bool {
	sig, err := btcec.ParseSignature(r.Sig, btcec.S256())
	if err != nil {
		return false
	}

	return sig.Verify(r.getSignContentHash(), pubKey)
}

func (r *Response) IsAccept() bool {
	return r.Accept == acceptFlag
}

func (r *Response) getSignContentHash() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, r.Version)
	binary.Write(buf, binary.BigEndian, r.Accept)
	binary.Write(buf, binary.BigEndian, r.CodeVersion)
	binary.Write(buf, binary.BigEndian, r.NodeType)
	binary.Write(buf, binary.BigEndian, r.SessionKey)

	result := utils.Hash(buf.Bytes())
	return result
}

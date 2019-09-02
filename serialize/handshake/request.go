package handshake

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/params"
	"github.com/996BC/996.Blockchain/utils"
)

type Request struct {
	Version     uint8
	ChainID     uint8
	CodeVersion params.CodeVersion
	NodeType    params.NodeType
	PubKey      []byte
	SessionKey  []byte
	Sig         []byte
}

func NewRequestV1(chainID uint8, codeVersion params.CodeVersion, nodeType params.NodeType,
	pubKey []byte, sessionKey []byte) *Request {
	return &Request{
		Version:     HandshakeV1,
		ChainID:     chainID,
		CodeVersion: codeVersion,
		NodeType:    nodeType,
		PubKey:      pubKey,
		SessionKey:  sessionKey,
	}
}

func UnmarshalRequest(data io.Reader) (*Request, error) {
	result := &Request{}
	var pubKeyLen uint8
	var sessionKeyLen uint8
	var sigLen uint16
	var err error

	if err = binary.Read(data, binary.BigEndian, &result.Version); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.ChainID); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.CodeVersion); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.NodeType); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &pubKeyLen); err != nil {
		return nil, err
	}
	result.PubKey = make([]byte, pubKeyLen)
	if err = binary.Read(data, binary.BigEndian, result.PubKey); err != nil {
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

func (r *Request) Marshal() []byte {
	result := new(bytes.Buffer)

	binary.Write(result, binary.BigEndian, r.Version)
	binary.Write(result, binary.BigEndian, r.ChainID)
	binary.Write(result, binary.BigEndian, r.CodeVersion)
	binary.Write(result, binary.BigEndian, r.NodeType)

	pubKeyLen := utils.Uint8Len(r.PubKey)
	binary.Write(result, binary.BigEndian, pubKeyLen)
	binary.Write(result, binary.BigEndian, r.PubKey)

	sessionKeyLen := utils.Uint8Len(r.SessionKey)
	binary.Write(result, binary.BigEndian, sessionKeyLen)
	binary.Write(result, binary.BigEndian, r.SessionKey)

	sigLen := utils.Uint16Len(r.Sig)
	binary.Write(result, binary.BigEndian, sigLen)
	binary.Write(result, binary.BigEndian, r.Sig)

	return result.Bytes()
}

// Sign generate the signature set to the field Sig
func (r *Request) Sign(privKey *btcec.PrivateKey) {
	sig, _ := privKey.Sign(r.getSignContentHash())
	r.Sig = sig.Serialize()
}

// Verify checks the response is valid or not
func (r *Request) Verify() bool {
	peerKey, err := btcec.ParsePubKey(r.PubKey, btcec.S256())
	if err != nil {
		return false
	}

	sig, err := btcec.ParseSignature(r.Sig, btcec.S256())
	if err != nil {
		return false
	}

	return sig.Verify(r.getSignContentHash(), peerKey)
}

func (r *Request) getSignContentHash() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, r.Version)
	binary.Write(buf, binary.BigEndian, r.ChainID)
	binary.Write(buf, binary.BigEndian, r.CodeVersion)
	binary.Write(buf, binary.BigEndian, r.NodeType)
	binary.Write(buf, binary.BigEndian, r.PubKey)
	binary.Write(buf, binary.BigEndian, r.SessionKey)

	result := utils.Hash(buf.Bytes())
	return result
}

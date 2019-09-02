package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"unicode/utf8"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	EvidenceMaxDescriptionLen = 140
	EvidenceBasicLen          = 1 + 4 + 1 + 1 + 2 + 2
)

type Evidence struct {
	Version     uint8
	Nonce       uint32
	Hash        []byte
	Description []byte
	PubKey      []byte
	Sig         []byte
	pc          *powCache
}

func NewEvidenceV1(hash, description, pubKey []byte) *Evidence {
	return &Evidence{
		Version:     CoreProtocolV1,
		Nonce:       0,
		Hash:        hash,
		Description: description,
		PubKey:      pubKey,
		Sig:         nil,
		pc:          newPowCache(),
	}
}

func UnmarshalEvidence(data io.Reader) (*Evidence, error) {
	result := &Evidence{}
	var hashLen uint8
	var descriptionLen uint16
	var pubKeyLen uint8
	var sigLen uint16
	var err error

	if err = binary.Read(data, binary.BigEndian, &result.Version); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &result.Nonce); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &hashLen); err != nil {
		return nil, err
	}
	result.Hash = make([]byte, hashLen)
	if err = binary.Read(data, binary.BigEndian, result.Hash); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &descriptionLen); err != nil {
		return nil, err
	}
	result.Description = make([]byte, descriptionLen)
	if err = binary.Read(data, binary.BigEndian, result.Description); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &pubKeyLen); err != nil {
		return nil, err
	}
	result.PubKey = make([]byte, pubKeyLen)
	if err = binary.Read(data, binary.BigEndian, result.PubKey); err != nil {
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

func (e *Evidence) Marshal() []byte {
	result := new(bytes.Buffer)

	binary.Write(result, binary.BigEndian, e.Version)
	binary.Write(result, binary.BigEndian, e.Nonce)

	hashLen := utils.Uint8Len(e.Hash)
	binary.Write(result, binary.BigEndian, hashLen)
	binary.Write(result, binary.BigEndian, e.Hash)

	descriptionLen := utils.Uint16Len(e.Description)
	binary.Write(result, binary.BigEndian, descriptionLen)
	binary.Write(result, binary.BigEndian, e.Description)

	pubKeyLen := utils.Uint8Len(e.PubKey)
	binary.Write(result, binary.BigEndian, pubKeyLen)
	binary.Write(result, binary.BigEndian, e.PubKey)

	sigLen := utils.Uint16Len(e.Sig)
	binary.Write(result, binary.BigEndian, sigLen)
	binary.Write(result, binary.BigEndian, e.Sig)

	return result.Bytes()
}

func (e *Evidence) SetNonce(nonce uint32) {
	e.Nonce = nonce
}

// NextNonce makes nonce++ and return pow value;
// the result is only readable, should not modify it
func (e *Evidence) NextNonce() *big.Int {
	if !e.pc.cacheBefore() {
		marshal := e.Marshal()
		pow := big.NewInt(0).SetBytes(utils.Hash(marshal))
		e.pc.setCache(marshal, pow)

		return pow
	}

	const nonceIndex = 1 // after Version
	e.Nonce++
	return e.pc.update(e.Nonce, nonceIndex)
}

func (e *Evidence) Verify() error {
	if e.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid evidence version %d", e.Version)
	}

	if len(e.Hash) != utils.HashLength {
		return fmt.Errorf("invalid hash length %d", len(e.Hash))
	}

	if len(e.PubKey) != btcec.PubKeyBytesLenCompressed {
		return fmt.Errorf("invalid public key length %d", len(e.PubKey))
	}

	if err := VerifyDescription(string(e.Description)); err != nil {
		return err
	}

	signature, err := btcec.ParseSignature(e.Sig, btcec.S256())
	if err != nil {
		return fmt.Errorf("invalid signature: %v", err)
	}

	key, err := btcec.ParsePubKey(e.PubKey, btcec.S256())
	if err != nil {
		return fmt.Errorf("invalid public key: %v", err)
	}

	if !e.verifySig(signature, key) {
		return fmt.Errorf("verify signature failed")
	}

	return nil
}

func (e *Evidence) Size() int {
	return EvidenceBasicLen + len(e.Hash) + len(e.PubKey) + len(e.Description) + len(e.Sig)
}

func (e *Evidence) GetSerializedHash() []byte {
	return utils.Hash(e.Marshal())
}

func (e *Evidence) GetPow() *big.Int {
	return big.NewInt(0).SetBytes(e.GetSerializedHash())
}

func (e *Evidence) String() string {
	return fmt.Sprintf("Hash %X PubKey %X Sig %X Nonce %d",
		e.Hash, e.PubKey, e.Sig, e.Nonce)
}

func (e *Evidence) Sign(key *btcec.PrivateKey) error {
	sig, err := key.Sign(e.getSignContentHash())
	if err != nil {
		return err
	}
	e.Sig = sig.Serialize()
	return nil
}

func (e *Evidence) verifySig(signature *btcec.Signature, key *btcec.PublicKey) bool {
	return signature.Verify(e.getSignContentHash(), key)
}

func (e *Evidence) getSignContentHash() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, e.Description)
	binary.Write(buf, binary.BigEndian, e.Hash)

	result := utils.Hash(buf.Bytes())
	return result
}

func VerifyDescription(d string) error {
	count := utf8.RuneCountInString(d)
	if count == -1 {
		return fmt.Errorf("invalid description, not utf-8 encoding")
	}
	if count > EvidenceMaxDescriptionLen {
		return fmt.Errorf("invalid description length %d", count)
	}

	return nil
}

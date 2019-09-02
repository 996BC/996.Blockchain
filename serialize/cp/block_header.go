package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/utils"
)

type BlockHeader struct {
	Version      uint8
	Time         int64
	Nonce        uint32
	Target       uint32
	LastHash     []byte
	Miner        []byte
	EvidenceRoot []byte
	pc           *powCache
}

func NewBlockHeaderV1(lastHash []byte, miner []byte, root []byte) *BlockHeader {
	return &BlockHeader{
		Version:      CoreProtocolV1,
		Time:         time.Now().Unix(),
		Nonce:        0,
		Target:       0,
		LastHash:     lastHash,
		Miner:        miner,
		EvidenceRoot: root,
		pc:           newPowCache(),
	}
}

func UnmarshalBlockHeader(data io.Reader) (*BlockHeader, error) {
	result := &BlockHeader{}
	var lastHashLen uint8
	var minerLen uint8
	var evRootLen uint8
	var err error

	if err = binary.Read(data, binary.BigEndian, &result.Version); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.Time); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.Nonce); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.Target); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &lastHashLen); err != nil {
		return nil, err
	}
	result.LastHash = make([]byte, lastHashLen)
	if err = binary.Read(data, binary.BigEndian, result.LastHash); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &minerLen); err != nil {
		return nil, err
	}
	result.Miner = make([]byte, minerLen)
	if err = binary.Read(data, binary.BigEndian, result.Miner); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &evRootLen); err != nil {
		return nil, err
	}
	result.EvidenceRoot = make([]byte, evRootLen)
	if err = binary.Read(data, binary.BigEndian, result.EvidenceRoot); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *BlockHeader) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, b.Version)
	binary.Write(result, binary.BigEndian, b.Time)
	binary.Write(result, binary.BigEndian, b.Nonce)
	binary.Write(result, binary.BigEndian, b.Target)

	lastHashLen := utils.Uint8Len(b.LastHash)
	binary.Write(result, binary.BigEndian, lastHashLen)
	binary.Write(result, binary.BigEndian, b.LastHash)

	minerLen := utils.Uint8Len(b.Miner)
	binary.Write(result, binary.BigEndian, minerLen)
	binary.Write(result, binary.BigEndian, b.Miner)

	evRootLen := utils.Uint8Len(b.EvidenceRoot)
	binary.Write(result, binary.BigEndian, evRootLen)
	binary.Write(result, binary.BigEndian, b.EvidenceRoot)

	return result.Bytes()
}

func (b *BlockHeader) UpdateTime() {
	b.Time = time.Now().Unix()
}

func (b *BlockHeader) ShallowCopy() *BlockHeader {
	return &BlockHeader{
		Version:      b.Version,
		Time:         b.Time,
		Nonce:        b.Nonce,
		Target:       b.Target,
		LastHash:     b.LastHash,
		Miner:        b.Miner,
		EvidenceRoot: b.EvidenceRoot,
		pc:           newPowCache(),
	}
}

func (b *BlockHeader) SetNonce(nonce uint32) {
	b.Nonce = nonce
}

func (b *BlockHeader) SetTarget(target uint32) {
	b.Target = target
}

// NextNonce makes nonce++ and return the pow value;
// the result is only readable, should not modify it
func (b *BlockHeader) NextNonce() *big.Int {
	if !b.pc.cacheBefore() {
		marshal := b.Marshal()
		pow := big.NewInt(0).SetBytes(utils.Hash(marshal))
		b.pc.setCache(marshal, pow)

		return pow
	}

	const nonceIndex = 1 + 8 // after Version and Time
	b.Nonce++
	return b.pc.update(b.Nonce, nonceIndex)
}

func (b *BlockHeader) Verify() error {
	if b.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid header version")
	}

	if len(b.LastHash) != utils.HashLength {
		return fmt.Errorf("invalid LastHash %X", b.LastHash)
	}

	if len(b.Miner) != btcec.PubKeyBytesLenCompressed {
		return fmt.Errorf("invalid Miner %X", b.Miner)
	}

	if b.EvidenceRoot == nil {
		return fmt.Errorf("nil EvidenceRoot")
	}

	if !bytes.Equal(b.EvidenceRoot, EmptyEvidenceRoot) && len(b.EvidenceRoot) != utils.HashLength {
		return fmt.Errorf("invalid EvidenceRoot %X", b.EvidenceRoot)
	}

	return nil
}

func (b *BlockHeader) GetSerializedHash() []byte {
	return utils.Hash(b.Marshal())
}

func (b *BlockHeader) GetPow() *big.Int {
	return big.NewInt(0).SetBytes(b.GetSerializedHash())
}

func (b *BlockHeader) IsEmptyEvidenceRoot() bool {
	return bytes.Equal(b.EvidenceRoot, EmptyEvidenceRoot)
}

func (b *BlockHeader) SetEmptyEvdsRoot() {
	b.EvidenceRoot = EmptyEvidenceRoot
}

func (b *BlockHeader) String() string {
	return fmt.Sprintf("Version %d Time %s Nonce %d Target %d LastHash %X Miner %X EvidenceRoot %X",
		b.Version, utils.TimeToString(b.Time), b.Nonce, b.Target, b.LastHash, b.Miner, b.EvidenceRoot)
}

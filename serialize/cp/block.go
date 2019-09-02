package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Block struct {
	*BlockHeader
	Evds []*Evidence
}

func NewBlock(header *BlockHeader, evds []*Evidence) *Block {
	return &Block{
		BlockHeader: header,
		Evds:        evds,
	}
}

func UnmarshalBlock(data io.Reader) (*Block, error) {
	result := &Block{}
	var evdsLen uint16
	var err error

	if result.BlockHeader, err = UnmarshalBlockHeader(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &evdsLen); err != nil {
		return nil, err
	}
	for i := uint16(0); i < evdsLen; i++ {
		var evd *Evidence
		if evd, err = UnmarshalEvidence(data); err != nil {
			return nil, err
		}
		result.Evds = append(result.Evds, evd)
	}

	return result, nil
}

func (b *Block) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, b.BlockHeader.Marshal())

	evidenceSize := uint16(len(b.Evds))
	binary.Write(result, binary.BigEndian, evidenceSize)
	for _, evd := range b.Evds {
		binary.Write(result, binary.BigEndian, evd.Marshal())
	}

	return result.Bytes()
}

func (b *Block) Verify() error {
	var err error

	if b.BlockHeader == nil {
		return fmt.Errorf("nil header")
	}

	if err = b.BlockHeader.Verify(); err != nil {
		return fmt.Errorf("block header verify failed:%v", err)
	}

	if b.IsEmptyEvidenceRoot() && len(b.Evds) != 0 {
		return fmt.Errorf("expect 0 evidence, but %d", len(b.Evds))
	}

	if !b.IsEmptyEvidenceRoot() && len(b.Evds) == 0 {
		return fmt.Errorf("expect evidence, but empty")
	}

	for _, evd := range b.Evds {
		if err = evd.Verify(); err != nil {
			return fmt.Errorf("evidence verify failed:%v", err)
		}
	}

	return nil
}

func (b *Block) ShallowCopy(onlyHeader bool) *Block {
	evds := b.Evds
	if onlyHeader {
		evds = nil
	}
	return &Block{
		BlockHeader: b.BlockHeader,
		Evds:        evds,
	}
}

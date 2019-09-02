package storage

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/996BC/996.Blockchain/serialize/cp"
)

type BlockHeader struct {
	*cp.BlockHeader
	Height uint64
}

func NewBlockHeader(h *cp.BlockHeader, height uint64) *BlockHeader {
	return &BlockHeader{
		BlockHeader: h,
		Height:      height,
	}
}

func UnmarshalBlockHeader(data io.Reader) (*BlockHeader, error) {
	result := &BlockHeader{}
	var err error

	if result.BlockHeader, err = cp.UnmarshalBlockHeader(data); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.Height); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *BlockHeader) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, b.BlockHeader.Marshal())
	binary.Write(result, binary.BigEndian, b.Height)
	return result.Bytes()
}

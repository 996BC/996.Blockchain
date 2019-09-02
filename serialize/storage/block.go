package storage

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

type Block struct {
	EvdsHash [][]byte
}

func NewBlock(evdsHash [][]byte) *Block {
	return &Block{
		EvdsHash: evdsHash,
	}
}

func UnmarshalBlock(data io.Reader) (*Block, error) {
	result := &Block{}
	var evdsSize uint16
	var err error

	if err = binary.Read(data, binary.BigEndian, &evdsSize); err != nil {
		return nil, err
	}

	for i := uint16(0); i < evdsSize; i++ {
		var hashLen uint8
		if err = binary.Read(data, binary.BigEndian, &hashLen); err != nil {
			return nil, err
		}

		hash := make([]byte, hashLen)
		if err = binary.Read(data, binary.BigEndian, hash); err != nil {
			return nil, err
		}
		result.EvdsHash = append(result.EvdsHash, hash)
	}

	return result, nil
}

func (b *Block) Marshal() []byte {
	result := new(bytes.Buffer)

	evdsSize := uint16(len(b.EvdsHash))
	binary.Write(result, binary.BigEndian, evdsSize)

	for _, evdH := range b.EvdsHash {
		hashLen := utils.Uint8Len(evdH)
		binary.Write(result, binary.BigEndian, hashLen)
		binary.Write(result, binary.BigEndian, evdH)
	}

	return result.Bytes()
}

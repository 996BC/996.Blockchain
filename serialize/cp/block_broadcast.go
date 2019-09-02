package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type BlockBroadcast struct {
	*Head
	Block *Block
}

func NewBlockBroadcast(block *Block) *BlockBroadcast {
	return &BlockBroadcast{
		Head:  NewHeadV1(MsgBlockBroadcast),
		Block: block,
	}
}

func UnmarshalBlockBroadcast(data io.Reader) (*BlockBroadcast, error) {
	result := &BlockBroadcast{}
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}
	if result.Block, err = UnmarshalBlock(data); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *BlockBroadcast) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, b.Head.Marshal())
	binary.Write(result, binary.BigEndian, b.Block.Marshal())
	return result.Bytes()
}

func (b *BlockBroadcast) Verify() error {
	if b.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid version %d", b.Version)
	}

	if b.Type != MsgBlockBroadcast {
		return fmt.Errorf("invlaid type %d", b.Type)
	}

	if b.Block == nil {
		return fmt.Errorf("nil block")
	}

	if err := b.Block.Verify(); err != nil {
		return fmt.Errorf("invalid block %v", err)
	}

	return nil
}

func (b *BlockBroadcast) String() string {
	return fmt.Sprintf("Block %X", b.Block.GetSerializedHash())
}

package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type BlockResponse struct {
	*Head
	Blocks []*Block
}

func NewBlockResponse(blocks []*Block) *BlockResponse {
	return &BlockResponse{
		Head:   NewHeadV1(MsgBlockResponse),
		Blocks: blocks,
	}
}

func UnmarshalBlockResponse(data io.Reader) (*BlockResponse, error) {
	result := &BlockResponse{}
	var blocksSize uint16
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &blocksSize); err != nil {
		return nil, err
	}
	for i := uint16(0); i < blocksSize; i++ {
		var block *Block
		if block, err = UnmarshalBlock(data); err != nil {
			return nil, err
		}
		result.Blocks = append(result.Blocks, block)
	}

	return result, nil
}

func (b *BlockResponse) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, b.Head.Marshal())

	blocksSize := uint16(len(b.Blocks))
	binary.Write(result, binary.BigEndian, blocksSize)
	for _, block := range b.Blocks {
		binary.Write(result, binary.BigEndian, block.Marshal())
	}

	return result.Bytes()
}

func (b *BlockResponse) Verify() error {
	if b.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid version %d", b.Version)
	}

	if b.Type != MsgBlockResponse {
		return fmt.Errorf("invalid type %d", b.Type)
	}

	for _, block := range b.Blocks {
		if err := block.Verify(); err != nil {
			return fmt.Errorf("invalid block %v", err)
		}
	}

	return nil
}

func (b *BlockResponse) String() string {
	return fmt.Sprintf("%d blocks", len(b.Blocks))
}

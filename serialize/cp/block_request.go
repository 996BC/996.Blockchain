package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

const (
	onlyHeaderFlag    = 1
	notOnlyHeaderFlag = 0
)

type BlockRequest struct {
	*Head
	Base       []byte
	End        []byte
	OnlyHeader uint8
}

func NewBlockRequest(base []byte, end []byte, onlyHeader bool) *BlockRequest {
	result := &BlockRequest{
		Head:       NewHeadV1(MsgBlockRequest),
		Base:       base,
		End:        end,
		OnlyHeader: notOnlyHeaderFlag,
	}
	if onlyHeader {
		result.OnlyHeader = onlyHeaderFlag
	}

	return result
}

func UnmarshalBlockRequest(data io.Reader) (*BlockRequest, error) {
	result := &BlockRequest{}
	var baseLen uint8
	var endLen uint8
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &baseLen); err != nil {
		return nil, err
	}
	result.Base = make([]byte, baseLen)
	if err = binary.Read(data, binary.BigEndian, result.Base); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &endLen); err != nil {
		return nil, err
	}
	result.End = make([]byte, endLen)
	if err = binary.Read(data, binary.BigEndian, result.End); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &result.OnlyHeader); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *BlockRequest) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, b.Head.Marshal())

	baseLen := utils.Uint8Len(b.Base)
	binary.Write(result, binary.BigEndian, baseLen)
	binary.Write(result, binary.BigEndian, b.Base)

	endLen := utils.Uint8Len(b.End)
	binary.Write(result, binary.BigEndian, endLen)
	binary.Write(result, binary.BigEndian, b.End)

	binary.Write(result, binary.BigEndian, b.OnlyHeader)
	return result.Bytes()
}

func (b *BlockRequest) Verify() error {
	if b.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid version %d", b.Version)
	}

	if b.Type != MsgBlockRequest {
		return fmt.Errorf("invalid type %d", b.Type)
	}

	if len(b.Base) != utils.HashLength {
		return fmt.Errorf("invalid base %X", b.Base)
	}

	if len(b.End) != utils.HashLength {
		return fmt.Errorf("invalid end %X", b.End)
	}

	return nil
}

func (b *BlockRequest) IsOnlyHeader() bool {
	return b.OnlyHeader == onlyHeaderFlag
}

func (b *BlockRequest) String() string {
	return fmt.Sprintf("Base %X End %X OnlyHeader %d", b.Base, b.End, b.OnlyHeader)
}

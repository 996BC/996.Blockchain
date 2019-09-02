package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

type SyncRequest struct {
	*Head
	Base []byte
}

func NewSyncRequest(base []byte) *SyncRequest {
	return &SyncRequest{
		Head: NewHeadV1(MsgSyncReq),
		Base: base,
	}
}

func UnmarshalSyncRequest(data io.Reader) (*SyncRequest, error) {
	result := &SyncRequest{}
	var baseLen uint8
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

	return result, nil
}

func (s *SyncRequest) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, s.Head.Marshal())

	baseLen := utils.Uint8Len(s.Base)
	binary.Write(result, binary.BigEndian, baseLen)
	binary.Write(result, binary.BigEndian, s.Base)

	return result.Bytes()
}

func (s *SyncRequest) Verify() error {
	if s.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid version %d", s.Version)
	}

	if s.Type != MsgSyncReq {
		return fmt.Errorf("invalid type %d", s.Type)
	}

	if len(s.Base) != utils.HashLength {
		return fmt.Errorf("invalid base %X", s.Base)
	}

	return nil
}

func (s *SyncRequest) String() string {
	return fmt.Sprintf("Base %X", s.Base)
}

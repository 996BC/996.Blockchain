package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

const (
	outOfDateFlag = 0
	uptodateFlag  = 1
)

type SyncResponse struct {
	*Head
	Base       []byte
	End        []byte
	HeightDiff uint32
	Uptodate   uint8
}

func NewSyncResponse(base []byte, end []byte, heightDiff uint32, uptodate bool) *SyncResponse {
	result := &SyncResponse{
		Head:       NewHeadV1(MsgSyncResp),
		Base:       base,
		End:        end,
		HeightDiff: heightDiff,
		Uptodate:   outOfDateFlag,
	}
	if uptodate {
		result.Uptodate = uptodateFlag
	}
	return result
}

func UnmarshalSyncResponse(data io.Reader) (*SyncResponse, error) {
	result := &SyncResponse{}
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

	if err = binary.Read(data, binary.BigEndian, &result.HeightDiff); err != nil {
		return nil, err
	}
	if err = binary.Read(data, binary.BigEndian, &result.Uptodate); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SyncResponse) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, s.Head.Marshal())

	baseLen := utils.Uint8Len(s.Base)
	binary.Write(result, binary.BigEndian, baseLen)
	binary.Write(result, binary.BigEndian, s.Base)

	endLen := utils.Uint8Len(s.End)
	binary.Write(result, binary.BigEndian, endLen)
	binary.Write(result, binary.BigEndian, s.End)

	binary.Write(result, binary.BigEndian, s.HeightDiff)
	binary.Write(result, binary.BigEndian, s.Uptodate)

	return result.Bytes()
}

func (s *SyncResponse) IsUptodate() bool {
	return s.Uptodate == uptodateFlag
}

func (s *SyncResponse) Verify() error {
	if s.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid version %d", s.Version)
	}

	if s.Type != MsgSyncResp {
		return fmt.Errorf("invalid type %d", s.Type)
	}

	if len(s.Base) != utils.HashLength {
		return fmt.Errorf("invalid base %X", s.Base)
	}

	if len(s.End) != utils.HashLength {
		return fmt.Errorf("invalid end %X", s.End)
	}

	return nil
}

func (s *SyncResponse) String() string {
	if s.IsUptodate() {
		return "already uptodate"
	}

	return fmt.Sprintf("Base %X End %X HeightDiff %d",
		s.Base, s.End, s.HeightDiff)
}

package discover

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/996BC/996.Blockchain/utils"
)

type Head struct {
	Version  uint8
	Type     DiscvMsgType
	Time     int64
	Reserved uint16
}

func NewHeadV1(t DiscvMsgType) *Head {
	return &Head{
		Version:  DiscoverV1,
		Type:     t,
		Time:     time.Now().Unix(),
		Reserved: uint16(0),
	}
}

func UnmarshalHead(data io.Reader) (*Head, error) {
	result := &Head{}
	if err := binary.Read(data, binary.BigEndian, &result.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(data, binary.BigEndian, &result.Type); err != nil {
		return nil, err
	}
	if err := binary.Read(data, binary.BigEndian, &result.Time); err != nil {
		return nil, err
	}
	if err := binary.Read(data, binary.BigEndian, &result.Reserved); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Head) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, h.Version)
	binary.Write(result, binary.BigEndian, h.Type)
	binary.Write(result, binary.BigEndian, h.Time)
	binary.Write(result, binary.BigEndian, h.Reserved)
	return result.Bytes()
}

func (h *Head) String() string {
	return fmt.Sprintf("Version %d Type %d Time %s",
		h.Version, h.Type, utils.TimeToString(h.Time))
}

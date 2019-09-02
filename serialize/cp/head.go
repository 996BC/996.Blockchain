package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Head struct {
	Version  uint8
	Type     CoreMsgType
	Reserved uint16
}

func NewHeadV1(t CoreMsgType) *Head {
	return &Head{
		Version:  CoreProtocolV1,
		Type:     t,
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
	if err := binary.Read(data, binary.BigEndian, &result.Reserved); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Head) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, h.Version)
	binary.Write(result, binary.BigEndian, h.Type)
	binary.Write(result, binary.BigEndian, h.Reserved)
	return result.Bytes()
}

func (h *Head) String() string {
	return fmt.Sprintf("Version %d Type %d", h.Version, h.Type)
}

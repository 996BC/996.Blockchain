package discover

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

// Head(12) | From(-) | To(-)
type Ping struct {
	*Head
	PubKey []byte
}

func NewPing(pubKey []byte) *Ping {
	return &Ping{
		Head:   NewHeadV1(MsgPing),
		PubKey: pubKey,
	}
}

func (p *Ping) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, p.Head.Marshal())

	pubKeyLen := utils.Uint8Len(p.PubKey)
	binary.Write(result, binary.BigEndian, pubKeyLen)
	binary.Write(result, binary.BigEndian, p.PubKey)

	return result.Bytes()
}

func (p *Ping) String() string {
	return fmt.Sprintf("Head %v PubKey %X", p.Head, p.PubKey)
}

func UnmarshalPing(data io.Reader) (*Ping, error) {
	result := &Ping{}
	var pubKeyLen uint8
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &pubKeyLen); err != nil {
		return nil, err
	}
	result.PubKey = make([]byte, pubKeyLen)
	if err = binary.Read(data, binary.BigEndian, result.PubKey); err != nil {
		return nil, err
	}

	return result, nil
}

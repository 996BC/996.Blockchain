package discover

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

type Pong struct {
	*Head
	PingHash []byte
	PubKey   []byte
}

func NewPong(pingHash []byte, pubKey []byte) *Pong {
	return &Pong{
		Head:     NewHeadV1(MsgPong),
		PingHash: pingHash,
		PubKey:   pubKey,
	}
}

func UnmarshalPong(data io.Reader) (*Pong, error) {
	result := &Pong{}
	var pingHashLen uint8
	var pubKeyLen uint8
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &pingHashLen); err != nil {
		return nil, err
	}
	result.PingHash = make([]byte, pingHashLen)
	if err = binary.Read(data, binary.BigEndian, result.PingHash); err != nil {
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

func (p *Pong) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, p.Head.Marshal())

	pingHashLen := utils.Uint8Len(p.PingHash)
	binary.Write(result, binary.BigEndian, pingHashLen)
	binary.Write(result, binary.BigEndian, p.PingHash)

	pubKeyLen := utils.Uint8Len(p.PubKey)
	binary.Write(result, binary.BigEndian, pubKeyLen)
	binary.Write(result, binary.BigEndian, p.PubKey)

	return result.Bytes()
}

func (p *Pong) String() string {
	return fmt.Sprintf("Head %v PingHash %X PubKey %X", p.Head, p.PingHash, p.PubKey)
}

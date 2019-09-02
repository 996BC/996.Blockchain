package discover

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

type GetNeighbours struct {
	*Head
	PubKey []byte
}

func NewGetNeighbours(pubKey []byte) *GetNeighbours {
	return &GetNeighbours{
		Head:   NewHeadV1(MsgGetNeighbours),
		PubKey: pubKey,
	}
}

func UnmarshalGetNeighbours(data io.Reader) (*GetNeighbours, error) {
	result := &GetNeighbours{}
	var pubKeyLength uint8
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err := binary.Read(data, binary.BigEndian, &pubKeyLength); err != nil {
		return nil, err
	}
	result.PubKey = make([]byte, pubKeyLength)
	if err := binary.Read(data, binary.BigEndian, result.PubKey); err != nil {
		return nil, err
	}

	return result, nil
}

func (g *GetNeighbours) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, g.Head.Marshal())
	binary.Write(result, binary.BigEndian, utils.Uint8Len(g.PubKey))
	binary.Write(result, binary.BigEndian, g.PubKey)
	return result.Bytes()
}

func (g *GetNeighbours) String() string {
	return fmt.Sprintf("Head %v PubKey %X\n", g.Head, g.PubKey)
}

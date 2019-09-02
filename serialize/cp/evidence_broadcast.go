package cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/996BC/996.Blockchain/utils"
)

type EvidenceBroadcast struct {
	*Head
	Evds []*Evidence
}

func NewEvidenceBroadcast(evds []*Evidence) *EvidenceBroadcast {
	return &EvidenceBroadcast{
		Head: NewHeadV1(MsgEvidenceBroadcast),
		Evds: evds,
	}
}

func UnmarshalEvidenceBroadcast(data io.Reader) (*EvidenceBroadcast, error) {
	result := &EvidenceBroadcast{}
	var evdsSize uint16
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &evdsSize); err != nil {
		return nil, err
	}
	for i := uint16(0); i < evdsSize; i++ {
		var evd *Evidence
		if evd, err = UnmarshalEvidence(data); err != nil {
			return nil, err
		}
		result.Evds = append(result.Evds, evd)
	}

	return result, nil
}

func (e *EvidenceBroadcast) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, e.Head)

	evdsSize := uint16(len(e.Evds))
	binary.Write(result, binary.BigEndian, evdsSize)
	for _, evd := range e.Evds {
		binary.Write(result, binary.BigEndian, evd.Marshal())
	}

	return result.Bytes()
}

func (e *EvidenceBroadcast) Verify() error {
	if e.Version != CoreProtocolV1 {
		return fmt.Errorf("invalid version %d", e.Version)
	}

	if e.Type != MsgEvidenceBroadcast {
		return fmt.Errorf("invalid type %d", e.Type)
	}

	if e.Evds == nil {
		return fmt.Errorf("nil Evds")
	}

	for _, evd := range e.Evds {
		if err := evd.Verify(); err != nil {
			return fmt.Errorf("invalid evidence:%v", err)
		}
	}

	return nil
}

func (e *EvidenceBroadcast) String() string {
	var hashPrefix string
	for _, evd := range e.Evds {
		hashPrefix += utils.ToHex(evd.Hash[:2]) + ","
	}
	hashPrefix = hashPrefix[:len(hashPrefix)-1]

	return fmt.Sprintf("%d evidence: %s", len(e.Evds), hashPrefix)
}

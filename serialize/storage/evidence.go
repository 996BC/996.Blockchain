package storage

import (
	"io"

	"github.com/996BC/996.Blockchain/serialize/cp"
)

type Evidence struct {
	*cp.Evidence
}

func UnmarshalEvidence(data io.Reader) (*Evidence, error) {
	result := &Evidence{}
	var err error

	if result.Evidence, err = cp.UnmarshalEvidence(data); err != nil {
		return nil, err
	}
	return result, nil
}

package blockchain

import (
	"fmt"
)

type ErrAlreadyUpToDate struct {
	reqHash []byte
}

func (a ErrAlreadyUpToDate) Error() string {
	return fmt.Sprintf("%X is already up to date", a.reqHash)
}

type ErrFlushingCache struct {
	reqHash []byte
}

func (f ErrFlushingCache) Error() string {
	return fmt.Sprintf("flushing happens while handling %X, give up", f.reqHash)
}

type ErrHashNotFound struct {
	reqHash []byte
}

func (s ErrHashNotFound) Error() string {
	return fmt.Sprintf("sync %X not found", s.reqHash)
}

type ErrInvalidBlockRange struct {
	info string
}

func (s ErrInvalidBlockRange) Error() string {
	return s.info
}

type ErrEvidenceAlreadyExist struct {
	evd []byte
}

func (e ErrEvidenceAlreadyExist) Error() string {
	return fmt.Sprintf("evidence %X exists", e.evd)
}

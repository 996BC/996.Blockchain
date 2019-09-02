package blockchain

import (
	"fmt"
)

type AlreadyUpToDate struct {
	reqHash []byte
}

func (a AlreadyUpToDate) Error() string {
	return fmt.Sprintf("%X is already up to date", a.reqHash)
}

type FlushCacheHappen struct {
	reqHash []byte
}

func (f FlushCacheHappen) Error() string {
	return fmt.Sprintf("flushing happens while handling %X, give up", f.reqHash)
}

type SyncHashNotFound struct {
	reqHash []byte
}

func (s SyncHashNotFound) Error() string {
	return fmt.Sprintf("sync %X not found", s.reqHash)
}

type SyncInvalidRequest struct {
	info string
}

func (s SyncInvalidRequest) Error() string {
	return s.info
}

type EvidenceAlreadyExist struct {
	evd []byte
}

func (e EvidenceAlreadyExist) Error() string {
	return fmt.Sprintf("evidence %X exists", e.evd)
}

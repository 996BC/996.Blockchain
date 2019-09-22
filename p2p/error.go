package p2p

import (
	"errors"
	"fmt"

	"github.com/996BC/996.Blockchain/params"
)

var ErrNegotiateInvalidSig = errors.New("invalid signature")

var ErrNegotiateConnectionRefused = errors.New("connection refused")

var ErrNegotiateChainIDMismatch = errors.New("chain id mismatch")

var ErrNegotiateNodeTypeMismatch = errors.New("node type mismatch: 1. the light nodes would not connect with each other; 2. the full nodes would not connect to the light nodes")

var ErrNegotiateTimeout = errors.New("timeout")

type ErrNegotiateCodeVersionMismatch struct {
	minimizeVersionRequired params.CodeVersion
	remoteVersion           params.CodeVersion
}

func (n ErrNegotiateCodeVersionMismatch) Error() string {
	return fmt.Sprintf("code version mismatch, minimize required %d, got %d",
		n.minimizeVersionRequired, n.remoteVersion)
}

type ErrNegotiateBrokenData struct {
	info string
}

func (n ErrNegotiateBrokenData) Error() string {
	return n.info
}

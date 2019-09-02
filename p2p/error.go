package p2p

import (
	"fmt"

	"github.com/996BC/996.Blockchain/params"
)

type NegotiateVerifySigFailed struct {
}

func (n NegotiateVerifySigFailed) Error() string {
	return "verify negotiate response signature failed"
}

type NegotiateGotRejection struct{}

func (n NegotiateGotRejection) Error() string {
	return "remote reject connection"
}

type NegotiateChainIDMismatch struct{}

func (n NegotiateChainIDMismatch) Error() string {
	return "chain id mismatch"
}

type NegotiateCodeVersionMismatch struct {
	minimizeVersionRequired params.CodeVersion
	remoteVersion           params.CodeVersion
}

func (n NegotiateCodeVersionMismatch) Error() string {
	return fmt.Sprintf("code version mismatch, minimize required %d, remote is %d",
		n.minimizeVersionRequired, n.remoteVersion)
}

type NegotiateNodeTypeMismatch struct {
}

func (n NegotiateNodeTypeMismatch) Error() string {
	return "light nodes would not connect with each other; full node would not connect to light node"
}

type NegotiateBrokenData struct {
	info string
}

func (n NegotiateBrokenData) Error() string {
	return n.info
}

type NegotiateTimeout struct {
}

func (n NegotiateTimeout) Error() string {
	return "negotiate timeout"
}

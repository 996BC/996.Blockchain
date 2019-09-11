package p2p

import (
	"errors"
	"fmt"
)

// Protocol is the interface that the p2p network protocols must implement
type Protocol interface {
	ID() uint8
	Name() string
}

// ProtocolRunner defines the interface for accessing the p2p network
type ProtocolRunner interface {
	// Send sends data to network
	// returns nil on success, or ErrPeerNotFound, ErrNoPeers on fail
	Send(dp *PeerData) error

	// GetRecvChan returns a channel for getting network data
	GetRecvChan() <-chan *PeerData
}

// PeerData is the data struct used in sending or receiving from netwoks
type PeerData struct {
	// the Peer is the send target or the receive source node ID
	// if it is an empty string, means broadcast to every nodes
	Peer string

	Data []byte
}

// ErrPeerNotFound means Peer not found
type ErrPeerNotFound struct {
	Peer string
}

func (p ErrPeerNotFound) Error() string {
	return fmt.Sprintf("Peer:%s not found", p.Peer)
}

// ErrNoPeers means don't find any peers on the network
var ErrNoPeers = errors.New("Not found any peers on the network yet")

//////////////////////////////////////////////////////////////////////////////////////
type protocolRunner struct {
	protocol Protocol
	Data     chan *PeerData
	sendFunc func(p Protocol, dp *PeerData) error
	n        *node
}

func newProtocolRunner(protocol Protocol, sendFunc func(p Protocol, dp *PeerData) error) *protocolRunner {
	runner := &protocolRunner{
		protocol: protocol,
		Data:     make(chan *PeerData, 2048),
		sendFunc: sendFunc,
	}
	return runner
}

func (p *protocolRunner) Send(dp *PeerData) error {
	return p.sendFunc(p.protocol, dp)
}

func (p *protocolRunner) GetRecvChan() <-chan *PeerData {
	return p.Data
}

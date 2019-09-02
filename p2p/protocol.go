package p2p

import "fmt"

// Protocol is the interface that the p2p network protocols must implement
type Protocol interface {
	ID() uint8
	Name() string
}

// ProtocolRunner defines the interface for accessing the p2p network
type ProtocolRunner interface {
	// Send sends data to network
	// returns nil on success, or PeerNotFoundError, NoPeersError on fail
	Send(dp *PeerData) error

	// GetRecvChan returns a channel for getting network data
	GetRecvChan() <-chan *PeerData
}

// PeerData combines network data and peer info
type PeerData struct {
	// the Peer is the send target or the receive source node ID;
	// if Peer is empty string, means broadcast to every nodes (used in sending data)
	Peer string
	Data []byte
}

// PeerNotFoundError means Peer not found
type PeerNotFoundError struct {
	Peer string
}

func (p PeerNotFoundError) Error() string {
	return fmt.Sprintf("Peer:%s not found", p.Peer)
}

// NoPeersError means don't find any peers on the network
type NoPeersError struct{}

func (p NoPeersError) Error() string {
	return "Not found any peers on the network yet"
}

//////////////////////////////////////////////////////////////////////////////////////
type protocolRunner struct {
	Data     chan *PeerData
	protocol Protocol
	n        *Node
}

func newProtocolRunner(protocol Protocol, node *Node) *protocolRunner {
	runner := &protocolRunner{
		protocol: protocol,
		n:        node,
		Data:     make(chan *PeerData, 2048),
	}
	return runner
}

func (p *protocolRunner) Send(dp *PeerData) error {
	return p.n.send(p.protocol, dp)
}

func (p *protocolRunner) GetRecvChan() <-chan *PeerData {
	return p.Data
}

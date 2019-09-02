package peer

import (
	"fmt"
	"net"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/crypto"
)

// Peer is a node that can connect to
type Peer struct {
	IP   net.IP
	Port int
	Key  *btcec.PublicKey

	// We use base32(compressed public key) as peer id
	// instead of others like base58(hash(public key)).
	// Since the hash is used to hide the real onwer
	// of the coins who never had sent any transactions,
	// and 996.Blockchain doesn't support transactions,
	// so the id is just another readable representation
	// of the public key.
	ID string
}

// NewPeer create a Peer, key might be nil if you don't know it
func NewPeer(ip net.IP, port int, key *btcec.PublicKey) *Peer {
	p := &Peer{
		IP:   ip,
		Port: port,
		Key:  key,
	}
	if key != nil {
		p.ID = crypto.PubKeyToID(key)
	}
	return p
}

func (p *Peer) String() string {
	return fmt.Sprintf("ID %s address %s", p.ID, p.Address())
}

// Address returns the peer ip address like 192.168.1.1:8080,[2001:0db8:85a3:08d3:1319:8a2e:0370:7344]:8443
func (p *Peer) Address() string {
	v4IP := p.IP.To4()
	if v4IP != nil {
		return fmt.Sprintf("%s:%d", v4IP.String(), p.Port)
	}
	return fmt.Sprintf("[%s]:%d", p.IP.String(), p.Port)
}

// Provider defines the interface for the peer provider
type Provider interface {
	Start()
	Stop()

	// GetPeers returns avaliable peers for the caller
	GetPeers(expect int, exclude map[string]bool) ([]*Peer, error)

	// AddSeeds adds seeds for provider's initilization
	// the seeds' Peer.Key should be nil
	AddSeeds(seeds []*Peer)
}

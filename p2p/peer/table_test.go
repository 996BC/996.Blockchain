package peer

import (
	"net"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/utils"
)

var tableTestVar = &struct {
	table            *tableImp
	seeds            []*Peer
	peers            []*Peer
	selfPeerIndex    int
	coolingPeerIndex int
	initSeedSize     int
	initPeerSize     int
}{
	seeds: []*Peer{
		NewPeer(net.ParseIP("192.168.1.1"), 10000, nil),
		NewPeer(net.ParseIP("192.168.1.2"), 10001, nil),
	},
}

func init() {
	tv := tableTestVar

	peerParams := []struct {
		keyHex string
		ip     net.IP
		port   int
	}{
		{"033ba26cfb499f20072ab448e195b759d3874dbb82839b03b07c687bf462589e74", net.ParseIP("192.168.2.1"), 10001},
		{"02d03652f539a447fe99780ab88607f419b1b0fd399d1fb8a7636237353cab53bc", net.ParseIP("192.168.2.2"), 10002},
		{"03c3972544c399570920265973bde43056d7a513afa00072e532d9a82a7ab743cd", net.ParseIP("192.168.2.3"), 10003},
		{"03f5641ee89f3ce15cb9df169d4dda6950633b015429b8762114d90c6eba204c17", net.ParseIP("192.168.2.4"), 10004},
	}

	for _, params := range peerParams {
		keyBytes, _ := utils.FromHex(params.keyHex)
		pubKey, _ := btcec.ParsePubKey(keyBytes, btcec.S256())
		peer := NewPeer(params.ip, params.port, pubKey)
		tv.peers = append(tv.peers, peer)
	}

	tv.selfPeerIndex = 0
	tv.coolingPeerIndex = 1
	tv.initSeedSize = len(tv.seeds)
	tv.initPeerSize = len(tv.peers) - 2 // exclude self and a cooling peer
}

func TestAddPeers(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	if err := utils.TCheckInt("table seeds size", len(tv.seeds), len(table.seeds)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("table peers size", tv.initPeerSize, len(table.peers)); err != nil {
		t.Fatal(err)
	}
}

func TestGetPeers(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	expectPeersSize := 0
	// all the peers are inactive
	for i := 0; i < tv.initPeerSize+1; i++ {
		peers := table.getPeers(i, nil)
		if err := utils.TCheckInt("get peers size", expectPeersSize, len(peers)); err != nil {
			t.Fatal(err)
		}
	}

	// all the peers are active
	for _, v := range table.peers {
		v.updateActiveTime()
	}
	for i := 0; i < tv.initPeerSize+1; i++ {
		if i < tv.initPeerSize {
			expectPeersSize = i
		} else {
			expectPeersSize = tv.initPeerSize
		}

		peers := table.getPeers(i, nil)
		if err := utils.TCheckInt("get peers size", i, len(peers)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetPeersWithExclude(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	// active all the peers
	for _, v := range table.peers {
		v.updateActiveTime()
	}

	peers := table.getPeers(tv.initPeerSize, nil)
	exclude := make(map[string]bool)
	exclude[peers[0].ID] = true

	peers = table.getPeers(tv.initPeerSize, exclude)
	if err := utils.TCheckInt("get peers size", tv.initPeerSize-1, len(peers)); err != nil {
		t.Fatal(err)
	}
}

func TestGetPeersToPing(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	expectSize := tv.initSeedSize + tv.initSeedSize

	// all the peers are timeout
	peers := table.getPeersToPing()
	if err := utils.TCheckInt("number of peers to ping", expectSize, len(peers)); err != nil {
		t.Fatal(err)
	}

	// set one of the peers to be active
	for _, v := range table.peers {
		// v.updateActiveTime()
		table.recvPong(v.Peer)
		break
	}
	peers = table.getPeersToPing()
	if err := utils.TCheckInt("number of peers to ping", expectSize-1, len(peers)); err != nil {
		t.Fatal(err)
	}
}

func TestGetPeersToGetNeighbours(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	peers := table.getPeersToGetNeighbours()
	if err := utils.TCheckInt("number of peers to get neighbours", tv.initPeerSize, len(peers)); err != nil {
		t.Fatal(err)
	}

	// get again should return no peers
	peers = table.getPeersToGetNeighbours()
	if err := utils.TCheckInt("number of peers to get neighbours", 0, len(peers)); err != nil {
		t.Fatal(err)
	}
}

func TestRecvPing(t *testing.T) {
	tv := tableTestVar

	// receive from new peer
	table := newTableImp()
	privKey, _ := btcec.NewPrivateKey(btcec.S256())
	pubKey := privKey.PubKey()
	peer := NewPeer(net.ParseIP("1.2.3.4"), 10000, pubKey)
	table.recvPing(peer)

	if err := utils.TCheckInt("peers size", tv.initPeerSize+1, len(table.peers)); err != nil {
		t.Fatal(err)
	}

	// receive from cooling peer
	table = newTableImp()
	table.recvPing(tv.peers[tv.coolingPeerIndex])
	if err := utils.TCheckInt("cooling peers size", 0, len(table.coolingPeers)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("peers size", tv.initPeerSize+1, len(table.peers)); err != nil {
		t.Fatal(err)
	}
}

func TestRecvPongFromSeed(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	privKey, _ := btcec.NewPrivateKey(btcec.S256())
	pubKey := privKey.PubKey()
	peer := NewPeer(tv.seeds[0].IP, tv.seeds[0].Port, pubKey)
	table.recvPong(peer)

	if err := utils.TCheckInt("seeds size", tv.initSeedSize-1, len(table.seeds)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("peers size", tv.initPeerSize+1, len(table.peers)); err != nil {
		t.Fatal(err)
	}

	for k, v := range table.peers {
		if k == crypto.PubKeyToID(pubKey) {
			if !v.isAvaible() {
				t.Fatal("expect pong peer active")
			} else {
				t.Log("update pong peer last active time successfully")
			}
		}
	}
}

func TestRecvPongFromPeer(t *testing.T) {
	table := newTableImp()

	var peer *Peer
	for _, v := range table.peers {
		peer = v.Peer
		break
	}
	table.recvPong(peer)

	for k, v := range table.peers {
		if k == peer.ID {
			if !v.isAvaible() {
				t.Fatal("expect pong peer active")
			} else {
				t.Log("update pong peer last active time successfully")
			}
		}
	}
}

func TestRefresh(t *testing.T) {
	tv := tableTestVar
	table := newTableImp()

	table.refresh()
	if err := utils.TCheckInt("cooling peers size", 0, len(table.coolingPeers)); err != nil {
		t.Fatal(err)
	}

	for _, v := range table.peers {
		v.lastActiveTime = time.Now().Add(-2 * peerExpiredTime)
		v.doPing()
		break
	}
	table.refresh()
	if err := utils.TCheckInt("peers size", tv.initPeerSize-1, len(table.peers)); err != nil {
		t.Fatal(err)
	}
}

// newTableImp returns a table with one cooling peer
// and its selfID is also from tableTestVar
func newTableImp() *tableImp {
	tv := tableTestVar

	// set the peer 0 as self
	selfID := tv.peers[tv.selfPeerIndex].ID
	t := newTable(selfID)
	tImp := t.(*tableImp)

	// set the peer 1 as cooling peer
	tImp.coolingPeers[tv.peers[tv.coolingPeerIndex].ID] = initTimepoint

	tImp.addPeers(tv.seeds, true)
	tImp.addPeers(tv.peers, false)

	return tImp
}

package discover

import (
	"bytes"
	"testing"

	"github.com/996BC/996.Blockchain/utils"
)

func verifyHead(t *testing.T, expect *Head, result *Head) {
	if err := utils.TCheckUint8("head version", expect.Version, result.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt64("head time", expect.Time, result.Time); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("head type", expect.Type, result.Type); err != nil {
		t.Fatal(err)
	}
}

func TestPing(t *testing.T) {
	pubKey := []byte("ping_pub_key")

	ping := NewPing(pubKey)
	pingBytes := ping.Marshal()

	rPing, err := UnmarshalPing(bytes.NewReader(pingBytes))
	if err != nil {
		t.Fatalf("unmarshal Ping failed:%v\n", err)
	}

	// verify
	verifyHead(t, ping.Head, rPing.Head)

	if err := utils.TCheckBytes("public key", pubKey, rPing.PubKey); err != nil {
		t.Fatal(err)
	}
}

func TestPong(t *testing.T) {
	hash := []byte("pong_used_ping_hash")
	pubKey := []byte("pong_public_key")

	pong := NewPong(hash, pubKey)
	pongBytes := pong.Marshal()

	rPong, err := UnmarshalPong(bytes.NewReader(pongBytes))
	if err != nil {
		t.Fatalf("unmarshal Pong failed:%v\n", err)
	}

	verifyHead(t, pong.Head, rPong.Head)
	if err := utils.TCheckBytes("ping hash", pong.PingHash, rPong.PingHash); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("public key", pong.PubKey, rPong.PubKey); err != nil {
		t.Fatal(err)
	}
}

func TestGetNeighbours(t *testing.T) {
	pubKey := []byte("get_neighbours_key")

	getNeighbours := NewGetNeighbours(pubKey)
	getNeighboursBytes := getNeighbours.Marshal()

	rGetNeighbours, err := UnmarshalGetNeighbours(bytes.NewReader(getNeighboursBytes))
	if err != nil {
		t.Fatalf("unmarshal GetNeighbours failed:%v\n", err)
	}

	verifyHead(t, getNeighbours.Head, rGetNeighbours.Head)
	if err := utils.TCheckBytes("public key", getNeighbours.PubKey, rGetNeighbours.PubKey); err != nil {
		t.Fatal(err)
	}
}

func TestNeighbours(t *testing.T) {
	// empty
	emptyNeighbour := NewNeighbours(nil)
	emptyNeighbourBytes := emptyNeighbour.Marshal()
	rEmptyNeighbour, err := UnmarshalNeighbours(bytes.NewReader(emptyNeighbourBytes))
	if err != nil {
		t.Fatalf("unmarshal empty Neighbours failed:%v\n", err)
	}
	verifyHead(t, emptyNeighbour.Head, rEmptyNeighbour.Head)

	if err := utils.TCheckInt("nodes number", 0, len(rEmptyNeighbour.Nodes)); err != nil {
		t.Fatal(err)
	}

	// with 2 nodes
	nodes := []*Node{
		NewNode(NewAddress("8.8.8.8", int32(10000)), []byte("Node_A_PubKey")),
		NewNode(NewAddress("6.6.6.6", int32(10080)), []byte("Node_B_PubKey")),
	}
	neighbours := NewNeighbours(nodes)
	neighboursBytes := neighbours.Marshal()
	rNeighbours, err := UnmarshalNeighbours(bytes.NewReader(neighboursBytes))
	if err != nil {
		t.Fatalf("unmarshal Neighbours failed:%v\n", err)
	}
	verifyHead(t, neighbours.Head, rNeighbours.Head)

	for i, node := range rNeighbours.Nodes {
		if err := utils.TCheckIP("neighbour ip", nodes[i].Addr.IP, node.Addr.IP); err != nil {
			t.Fatal(err)
		}
		if err := utils.TCheckInt32("neighbour port", nodes[i].Addr.Port, node.Addr.Port); err != nil {
			t.Fatal(err)
		}
		if err := utils.TCheckBytes("neighbour public key", nodes[i].PubKey, node.PubKey); err != nil {
			t.Fatal(err)
		}
	}
}

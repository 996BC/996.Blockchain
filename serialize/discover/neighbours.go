package discover

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Neighbours struct {
	*Head
	Nodes []*Node
}

func NewNeighbours(nodes []*Node) *Neighbours {
	return &Neighbours{
		Head:  NewHeadV1(MsgNeighbours),
		Nodes: nodes,
	}
}

func UnmarshalNeighbours(data io.Reader) (*Neighbours, error) {
	result := &Neighbours{}
	var nodesNum uint16
	var nodes []*Node
	var err error

	if result.Head, err = UnmarshalHead(data); err != nil {
		return nil, err
	}

	if err = binary.Read(data, binary.BigEndian, &nodesNum); err != nil {
		return nil, err
	}
	for i := uint16(0); i < nodesNum; i++ {
		var node *Node
		if node, err = UnmarshalNode(data); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	result.Nodes = nodes

	return result, nil
}

func (n *Neighbours) Marshal() []byte {
	result := new(bytes.Buffer)
	binary.Write(result, binary.BigEndian, n.Head.Marshal())

	nodesNum := uint16(len(n.Nodes))
	binary.Write(result, binary.BigEndian, nodesNum)

	for i := uint16(0); i < nodesNum; i++ {
		binary.Write(result, binary.BigEndian, n.Nodes[i].Marshal())
	}
	return result.Bytes()
}

func (n *Neighbours) String() string {
	result := fmt.Sprintf("Head %v", n.Head)
	for i, node := range n.Nodes {
		result += fmt.Sprintf("[%d] %v", i, node)
	}
	return result
}

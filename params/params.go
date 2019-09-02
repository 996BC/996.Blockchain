package params

type NodeType = uint8

const (
	FullNode  = NodeType(1)
	LightNode = NodeType(2)
)

/////////////////////////////////////////////////////////////////

type CodeVersion uint16

const (
	// NodeVersionV1 starts from v1.0.0
	NodeVersionV1 = CodeVersion(1)
)

var CurrentCodeVersion = NodeVersionV1
var MinimizeVersionRequired = NodeVersionV1

////////////////////////////////////////////////////////////////

const (
	// BlockSize is 1MB
	BlockSize = 1024 * 1024
)

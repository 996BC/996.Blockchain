package handshake

import (
	"bytes"
	"sync"
)

const (
	// HandshakeV1 (handshake vesion 1)
	HandshakeV1 = 1
)

var hsSigContentBufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

/*
Request
+---------------+---------------+-------------------+------------+
|   Version     |   ChainID     |   CodeVersion     |   NodeType |
+---------+-----+---------------+-------------------+------------+
| PubKeyL |                     PubKey                           |
+---------+---+--------------------------------------------------+
| SessionKeyL |                 SessionKey                       |
+------+------+--------------------------------------------------+
| SigL |                        Sig                              |
+------+---------------------------------------------------------+

(bytes)
Version             1
ChainID             1
CodeVersion         2
NodeType            1
PubKey length       1
PubKey              -
SessionKey length   1
SessionKey          -
Sig lenghth         2
Sig                 -

Response
+---------------+---------------+-------------------+------------+
|   Version     |   Accept      |   CodeVersion     |   NodeType |
+-------------+-+---------------+-------------------+------------+
| SessionKeyL |                 SessionKey                       |
+------+------+--------------------------------------------------+
| SigL |                        Sig                              |
+------+---------------------------------------------------------+

(bytes)
Version             1
Accept              1
CodeVersion         2
NodeType            1
SessionKey length   1
SessionKey          -
Sig lenghth         2
Sig                 -
*/

package cp

type CoreMsgType = uint8

const (
	// CoreProtocolV1 is the version 1 of the core protocol
	CoreProtocolV1 = 1

	// core message type
	MsgSyncReq           = 1
	MsgSyncResp          = 2
	MsgBlockRequest      = 3
	MsgBlockResponse     = 4
	MsgBlockBroadcast    = 5
	MsgEvidenceBroadcast = 6
)

var (
	// EmptyEvidenceRoot is used in the BlockHeader's field
	// It means the block contains no evidence
	EmptyEvidenceRoot = []byte("EMPTY")
)

/*

Evidence
+---------------------------------+
|            Version              |
+---------------------------------+
|             Nonce               |
+---------+-----------------------+
| HashL   |         Hash          |
+---------+-----+-----------------+
|DescriptionL   |   Description   |
+---------+-----+-----------------+
| PubKeyL |        PubKey         |
+---------+-----+-----------------+
| SigL          |     Sig         |
+---------------+-----------------+
(bytes)
Version             1
Nonce               4
Hash length         1
Hash                -
PubKey length       1
PubKey              -
Description length  2
Description         -
Sig length          2
Sig                 -


BlockHeader
+---------+------+-------+--------+
| Version | Time | Nonce | Target |
+---------+-+----+-------+--------+
| LastHashL |     LastHash        |
+-----------+---------------------+
| MinerL    |     Miner           |
+-----------+---+-----------------+
| EvidenceRootL | EvidenceRoot    |
+---------------+-----------------+
(bytes)
Version                     1
Time                        8
Nonce                       4
Target                      4
LastHash length             1
LastHash                    -
Miner length                1
Miner                       -
EvidenceRoot length         1
EvidenceRoot                -


Block
+-----------------------------+
|         (BlockHeader)       |
+-----------+-----------------+
| Evds size | Evds:(Evidence) |
+-----------+-----------------+
(bytes)
Evds size       2
Evds            sizeof(Evidence) * Evds size

Head
+---------+------+----------+
| Version | Type | Reserved |
+---------+------+----------+
(bytes)
Version     1
Type        1
Reserved    2


SyncRequest
+-----------------------------+
|           (Head)            |
+--------+--------------------+
| BaseL  |      Base          |
+--------+--------------------+
(bytes)
Base length     1
Base            -


SyncResponse
+-----------------------------+
|           (Head)            |
+--------+--------------------+
| BaseL  |      Base          |
+--------+--------------------+
| EndL   |      End           |
+--------+----+---------------+
| HeightDiff  |   Uptodate    |
+-------------+---------------+
(bytes)
Base length     1
Base            -
End length      1
End             -
HeightDiff      4
Uptodate        1


BlockRequest
+-----------------------------+
|           (Head)            |
+--------+--------------------+
| BaseL  |      Base          |
+--------+--------------------+
| EndL   |      End           |
+--------+--------------------+
|         onlyHeader          |
+-----------------------------+
(bytes)
Base length         1
Base                -
End length          1
End                 -
onlyHeader          1


BlockResponse
+------------------------------+
|           (Header)           |
+-------------+----------------+
| Blocks size | Blocks:(Block) |
+-------------+----------------+
(bytes)
Blocks size     2
Blocks          sizeof(Block) * Blocks size


BlockBroadcast
+-----------------------------+
|           (Head)            |
+-----------------------------+
|        Block:(Block)        |
+-----------------------------+


EvidenceBroadcast
+-----------------------------+
|           (Head)            |
+-----------------------------+
| Evds size | Evds:(Evidence) |
+-----------+-----------------+
(bytes)
Evds size       2
Evds            sizeof(Evidence) * Evds size
*/

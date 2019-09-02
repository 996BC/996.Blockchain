package discover

type DiscvMsgType = uint8

const (
    // DiscoverV1 is the version 1 of the discover protocol
    DiscoverV1 = 1

    // discover message type
    MsgPing          = DiscvMsgType(1)
    MsgPong          = DiscvMsgType(2)
    MsgGetNeighbours = DiscvMsgType(3)
    MsgNeighbours    = DiscvMsgType(4)
)

/*
Head
+---------+------+--------+----------+
| Version | Type |  Time  | Reserved |
+---------+------+--------+----------+
(bytes)
Version     1
Type        1
Time        8
Reserved    2


Address
+-----------+--------+------+
| IP length |   IP   | Port |
+-----------+--------+------+
(bytes)
IP length   1
IP          -
Port        8


Node
+---------------------------+
|        (Address)          |
+---------+-----------------+
| PubKeyL |     PubKey      |
+---------+-----------------+
(bytes)
PubKey length   1
PubKey          -


Ping
+---------------------------+
|           (Head)          |
+---------+-----------------+
| PubKeyL |     PubKey      |
+---------------------------+
(bytes)
PubKey length   1
PubKey          -


Pong
+---------------------------+
|           (Head)          |
+---------------------------+
| PingHashL |   PingHash    |
+-----------+---------------+
| PubKeyL   |   PubKey      |
+---------+-----------------+
(bytes)
PingHash length     1
PingHash            -
PubKey length       1
PubKey              -


GetNeighbours
+---------------------------+
|           (Head)          |
+---------------------------+
| PubKeyL |     PubKey      |
+---------+-----------------+
(bytes)
PubKey length       1
PubKey              -


Neighboures
+---------------------------+
|           (Head)          |
+---------------------------+
|       Nodes size          |
+---------------------------+
|       Nodes:(Node)        |
+---------------------------+
(bytes)
Nodes size      2
Nodes           sizeof(Node) * Nodes size
*/

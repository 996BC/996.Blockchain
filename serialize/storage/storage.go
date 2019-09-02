package storage

/*
BlockHeader
+-----------------------+
|    (cp.BlockHeader)   |
+-----------------------+
|        Height         |
+-----------------------+
(bytes)
Height          8


Block
+-----------------------+
|  Evidence hash size   |
+-------+---------------+
| HashL |    Hash       |
+-------+---------------+
|        ......         |
+-----------------------+
(bytes)
Evidence hash size      2
Hash length             1
Hash                    -


Evidence
+-----------------------+
|    (cp.Evidence)      |
+-----------------------+

*/

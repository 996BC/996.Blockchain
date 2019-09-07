# 协议与接口

## 目录
* [基础信息](#基础信息)
    * [模块](#模块)
    * [参数](#参数)
* [数据格式与协议](#数据格式与协议)
    * [P2P发现](#p2p发现)
    * [握手](#握手)
    * [coreProtocol](#coreprotocol)
        * [数据结构](#数据结构)
        * [协议](#协议)
* [HTTP接口](#http接口)
    * [基本规则](#基本规则)
        * [返回值](#返回值)
    * [证据](#证据)
        * [上传证据](#上传证据)
        * [上传未签名的证据](#上传未签名的证据)
        * [查询证据](#查询证据)
    * [区块](#区块)
        * [通过高度范围查询区块](#通过高度范围查询区块)
        * [通过哈希查询区块](#通过哈希查询区块)
    * [账户](#账户)
        * [通过ID查询账户](#通过id查询账户)


## 基础信息

### 模块

下面描述的模块对应着golang的模块，和目录名一致。  

模块 | 描述
--- | ---
cmd | 各程序的入口
core | 挖矿、共识的核心模块
crypto | 密钥管理，密钥和ID的映射关系定义
db  | 数据持久化存储
p2p | 节点发现、节点连接，为上层协议提供抽象接口
rpc | 对外提供的HTTP查询接口
serialize | 定义各类可序列化和反序列化的数据，用于存储、网络协议
utils | 杂项

### 参数

参数 | 值 | 描述
--- | --- | ---
主网ID | 1 | -
块大小 | 1MB | -
出块时间　| 90秒　| -
证据大小　| 约 150~700 Bytes | 上传的哈希描述会影响大小，最多可填140个字符
哈希长度　| 32 Bytes | 一律使用sha256
最低的出块POW　| E8100000 | 0x100000 << (0xE8 - 24),低于这个值时取这个值
最低的证据POW | EE100000 | 0x100000 << (0xEE - 24)，低于这个值的证据不会被打包入块
难度调整参考区块数量 | 20 | 调整系数为 (前20个块出块时间 * 0.9 +　当前距离上一个块时间 * 0.1) / 20个块预期时间
账户ID | - | 账户压缩公钥的base32编码(不填充)

## 数据格式与协议

下文只对数据格式和协议作用做基本的描述，不深入讨论细节。

### P2P发现

具体格式参考[源码注释](/serialize/discover/discover.go)　/serialize/discover/discover.go

类型 | 值 | 描述
--- | --- | ---
Ping | 1 | 心跳探测
Pong | 2 | Ping应答
GetNeighbours | 3 | 获取邻居节点请求 
Neighbours | 4 | GetNeighbours应答，或主动推送邻居节点

* 每个节点维护着邻居表
* 定期向邻居发送Ping测试对端存活，长期未收到Pong响应会剔除相应节点
* 定期向邻居发送GetNeighbours请求，将Neighbours相应中带的节点加入到自己的邻居表

### 握手

具体格式参考[源码注释](/serialize/handshake/handshake.go) /serialize/handshake/handshake.go

**Request类型**

字段　| 字段含义　
--- | ---
Version | 握手协议版本
ChainID | 链ID
CodeVersion | 代码版本
NodeType | 节点类型
PubKey | 请求方公钥
SessionKey | 请求方会话公钥
Sig | 签名

**Response类型**

字段　| 字段含义　
--- | ---
Version | 握手协议版本
Accept | 是否接受
CodeVersion | 代码版本
NodeType  | 节点类型
SessionKey | 响应方会话公钥
Sig | 签名

* 使用椭圆曲线secp256k1进行签名和密钥协商
* 使用sha512做KDF，结果前32位为会话密钥，接下来的12位做随机值
* 使用AES-256-GCM进行后续的加密通信

### coreProtocol

具体格式参考[源码注释](/serialize/cp/cp.go)　/serialize/cp/cp.go

#### 数据结构

数据 | 描述
--- | ---
Evidence | 表示证据的结构，做pow时基于该格式的序列化结果进行
BlockHeader | 表示区块头，做pow时基于该格式的序列化结果进行
Block | 区块，是上述两者的组合，用于网络传输

#### 协议

类型 | 值 | 描述
--- | --- | ---
SyncReq | 1 | 同步请求，节点会带上当前最新的区块哈希
SyncResp | 2 | SyncReq响应，如果发现对端区块不是最新，会返回最新区块哈希以及两者间的高度差
BlockRequest | 3 | 根据SyncResp生成区块请求，期待获取某个区块哈希间的所有块
BlockResponse | 4 | BlockRequest响应，可生成多个响应，每次最大携带16个区块
BlockBroadcast | 5 | 区块广播
EvidenceBroadcast | 6 | 证据广播

* 节点起来后会发送SyncReq，根据响应生成BlockRequest拉取区块，直到所有的SyncResp都告知已经最新时才停止初始化同步
* 当进行着区块拉取时，不会再发同步请求，因此对BlockResponse有超时限制，目前期待每个块的网络传输时延为5s
* 运行期间节点也会定时向网络查询最新信息
* 当节点第一次收到广播后会广播给其他节点并记录下广播内容，下次收到同样广播时不再进行处理

## HTTP接口

anti996 运行期间会监听本地端口(默认23666)提供HTTP服务，client 的部分功能是基于这些接口实现的。

### 基本规则

规则 | 描述
--- | ---
URL | /版本号/业务/操作，如 /v1/block/query-via-range 表示通过高度范围查询区块
方法 | 只有GET和POST
编码 | UTF-8
返回值　| 见下文

#### 返回值

HTTP请求正常时都返回200，并附带以下格式的应答:
```json
{
    "code":0,
    "msg":"",
    "data":""
}
```

字段 | 描述
--- | ---
code | 返回码，成功返回0，失败返回1，请求参数有误返回2 
msg | 返回的消息，当code为1时不为空
data | 每个接口返回的响应数据，对应下文有返回数据的接口里的data字段

### 证据

#### 上传证据

**POST /v1/evidence/upload**

**请求结构** 
```json
# data数组中的每一项均表示一条证据
{
    "data": [
        {
            "version": 1,
            "hash": "xxxx",
            "description": "xxxx",
            "public_key": "xxxx",
            "sigature": "xxxx",
            "nonce": 10000
        }
    ]
}
```

字段 | 描述
--- | ---
version | 证据版本，当前均为１
hash | 证据摘要，十六进制编码
description | 证据的描述，不超过140个字符，可为空
public_key | 证据持有者公钥，十六进制编码
sigature | 签名信息，十六进制编码
nonce | 随机值,需按照3.3节的Evidence序列化格式进行POW得出

**响应结构**
无

#### 上传未签名的证据

请求节点为你进行POW并签名，确保节点所持私钥是自己的私钥。

**POST /v1/evidence/upload-raw**

**请求结构** 
```json
# data数组中的每一项均表示一条证据
{
   "evds":[
      {
         "hash":"xxx",
         "description":"yyy"
      }
   ]
}
```

字段 | 描述
--- | ---
hash | 证据摘要，十六进制编码
description | 对证据的描述，不超过140个字符，可为空

**响应结构**
无

#### 查询证据

**POST /v1/evidence/query**

**请求结构** 
```json
# data数组中每一项表示一条证据的哈希
{
    "hash":["xxxx", "xxxx", "xxxx"]
}
```

字段 | 描述
--- | ---
hash | 需要查询的证据的哈希值集合，哈希值使用十六进制编码

**响应结构** data数组中每一项均表示一条证据
```json
{
    "data": [
        {
            "version": 1,
            "hash": "xxxx",
            "description": "xxxx",
            "public_key": "xxxx",
            "sigature": "xxxx",
            "nonce": 10000,
            "height": 10000,
            "block_hash": "xxxx",
            "time": 123456,
        }
    ]
}
```

字段 | 描述
--- | ---
version | 证据版本
hash | 证据摘要，十六进制编码
description | 证据的描述
public_key | 证据持有者公钥，十六进制编码
sigature | 签名信息，十六进制编码
nonce | 随机值
height | 所在区块高度
block_hash | 所在区块的哈希值，十六进制编码
time | 所在区块的时间，1970/1/1至今的秒数

### 区块

区块查询的**返回结构**如下所示，其中data数组中的每一项表示一个区块，evds数组中的每一项表示该区块包含的证据信息。下面小节不再赘述该结构。

```json
{
    "data":[
        {
            "version": 1,
            "time": 123456,
            "nonce": 10000,
            "target": 10000,
            "last_hash": "xxxx",
            "miner": "xxxx",
            "evidence_root": "xxxx",
            "height": 100,
            "hash": "xxxx",
            "evds":[
                {
                    "hash":"xxxx",
                    "owner":"xxxx"
                }
            ]
        }
    ]
}
```
区块字段 | 描述
--- | ---
version | 区块版本
time | 所在区块的时间，1970/1/1至今的秒数
nonce | 随机值
target | 难度
last_hash | 上一个块的哈希值，十六进制编码
miner | 挖出该块的矿工公钥，十六进制编码
evidence_root | 证据的默克树根，十六进制编码
height | 区块高度
hash | 区块的哈希值，十六进制编码

证据字段 | 描述
--- | ---
hash | 证据的哈希值，十六进制编码
owner | 所有者公钥，十六进制编码

#### 通过高度范围查询区块

**GET /v1/block/query-via-range?range=...**

请求参数格式 |　描述
--- | ---
x | 指定某个高度ｘ
x,y,z | 指定多个高度，逗号分隔
ｘ-y | 从高度x至y，如1-100表示查询高度1-100的区块，如果超出范围，则只返回范围内的区块
-1 | 最新的区块


#### 通过哈希查询区块

**GET /v1/block/query-via-hash?hash=...**

请求参数格式 |　描述
--- | ---
x | 指定某个哈希值查询，需要十六进制编码
x,y,z | 指定多个哈希值查询，需要十六进制编码

### 账户

#### 通过ID查询账户

**GET /v1/account/query?id=...**

请求参数格式 |　描述
--- | ---
x | 用户ID，压缩公钥的base32编码(不填充)

**返回结构**
```json
{
    "data":{
        "evidence":["xxx","yyy","zzz"],
        "score":0
    }
}
```

字段 | 描述
--- | ---
evidence | 该账户所持的证据哈希，十六进制编码
score | 该账户的挖矿得分，没挖出一个块计1分(该数据并不存在链上，只从链上统计而得)

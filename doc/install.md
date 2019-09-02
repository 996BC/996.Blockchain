# 安装指引

## 依赖

### 1. golang >= 1.12

### 2. 设置goproxy(可选)

编译依赖 golang.org/x/ 的包，可能你的机器无法访问该网址而导致下载失败，可通过设置代理解决，修改.profile加入以下两行

```
export GO111MODULE=on
export GOPROXY=https://goproxy.io
```

source ~/.profie  使修改生效

## 安装

```shell
git clone https://github.com/996BC/996.Blockchain.git
cd 996.Blockchain
./build
```

等待完成以后，在bin目录下可以看到四个可执行程序 

程序名 | 作用
--- | --
anti996 | 节点程序
client | 客户端，主要用于生成、上传、查询摘要
dbbrowser | 落地磁盘数据的查询工具
keygen | 密钥的生成、转换工具

如果你是初次使用，应该先用 keygen 生成自己的私钥，然后运行 anti996 加入网络，当需要上传证据或查看区块数据时使用 client 与本地 anti996 通信。如果 anti996 不在运行，又想查询本地磁盘上的区块信息，可以使用 dbbrowser 。

详情参考[操作指引](/doc/instructions.md)




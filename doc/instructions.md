# 操作指引

下文会分别介绍四个编译出来的文件keygen, anti996, client, dbbrowser，并在最后一部分介绍两个完整的例子。所有工具都有-h指令，下文不再描述。

* [keygen](#kengen)
* [anti996](#anti996)
* [client](#client)
* [dbbrowser](#dbbrowser)
* [示例一: 运行anti996接入主网](#示例一-运行anti996接入主网)
* [示例二: 上传证据到网络中](#示例二-上传证据到网络中)


## keygen

keygen 是一个用于生成密钥和密钥转换的工具，密钥是用户身份的唯一标识，密钥文件不能泄露给任何人。anti996和client运行时都需要密钥信息，且只认名为.pKey和.sKey的文件，所有关于密钥的路径配置都是密钥的目录，无需指定文件名。  
  
密钥类型 | 介绍 | 文件名　| 补充
--- | --- | --- | ---
pKey(plain key) | 256位密钥的16进制表示 | .pKey | 
sKey(sealed key) | 密钥的内容经过用户的自定义密码加密 | .sKey | 每次使用时都需要输入解密密码
  
用户可以根据需要把pKey和sKey互相转换，也可以更改自定义的密码(根据sKey生成新的sKey)。  

指令 | 介绍
--- | ---
-m | 工作模式，有生成sKey,生成pKey，pKey转sKey，sKey转pKey，sKey转新的sKey
-o | 输出目录
-s | 输入文件，仅当转换时需要指明源文件

```shell
# 示例
# 在当前目录生成pKey
./keygen -m 2 -o .

# 把当前目录的pKey转换成sKey，并保存在当前目录
./keygen -m 4 -s . -o .
```

## anti996

anti996 是核心程序，和网络中的其他节点一起生成、同步区块，运行**需要 -c 参数**用于指定配置文件，配置文件参考项目根目录中的[config.json](/config.json)，配置的含义可以参考根目录中的[config.README](/config.README)。

注意：**如果运行程序接入主网，chain_id、难度设置、区块间隔、genesis这些共识基本配置不应修改。**

## client

client 是和整个网络通信的客户端工具，运行时也需要指定配置文件，默认会读取当前目录下的配置文件，配置文件参考项目 cmd/client/ 目录下的 config.json 文件，配置的含义可以参考同目录的config.README。client运行时必须指定配置文件，如果涉及到网络操作，则需要其指定的anti996服务端也在运行。

指令 | 介绍
--- | ---
-c | 指定配置文件，默认是 ./config.json
-e | 指定需要生成hash摘要的文件夹或文件，结果会以文件形式保存在当前的运行目录，如果目标是文件foo，则生成名为 hf-foo-{timestamp} 的文件，其中{timestamp}是精确到秒的时间戳
-qa | 查询账户信息，包括账户上链的hash和它的得分
-qb | 查询指定高度的区块信息，支持三种格式:"1,2,100"、"1-100"、"-1"，最后一种表示最新的区块
-qe | 查询的证据，查询多个可用逗号分割
-u | 把 -e 生成的结果上传到链上，此时会用账户对文件内的根哈希进行签名，并进行POW
-m | 描述hash含义，140个字符长度,utf8编码，一般上传证据时使用

## dbbrowser 

dbbrowser 用于查看已经落地的区块数据，需要指定数据库目录，注意该目录只能被一个运行实例锁定，所以anti996 和 dbbrowser不能同时运行（一般情况下anti996运行时通过client来查看区块数据）。

指令 | 介绍
--- | ---
-dbpath | 指定数据库目录
-b | 查询指定的区块信息
-e | 查询指定的证据信息
-range | 根据高度范围查询区块信息
-o | 把结果输出到指定文件

```shell
# 示例

# 查询1-20的区块信息并存到文件中
./dbbrowser -dbpath /your/path/to/database -range 1-20 -o blocks_info.txt

# 查询指定哈希值为C539D65656959805F4A4648CCD290D1E899BB9E83AA29244E8981CC8401AE310的证据
./dbbrowser -dbpath /your/path/to/database -e C539D65656959805F4A4648CCD290D1E899BB9E83AA29244E8981CC8401AE310
```

## 示例一: 运行anti996接入主网

```shell
# 编译
./build.sh

# 完成后，比如想在home目录的anti996目录中管理一切，则
mkdir -p ~/anti996 && cp ./bin/anti996 ./config.json ~/anti996

# 生成密钥存到anti996目录中
./bin/keygen -m 2 -o ~/anti996

# 进入到运行目录，建一个数据库需要的data目录
cd ~/anti996 && mkdir -p data

# 运行anti996 
nohup ./anti996 -c config.json 2>&1 >./log &

# 此时如果不出意外已经在运行了，节点会从网络上的其他节点同步区块信息，保存在data目录中，同步完成后开始挖矿
# 如果不希望自己的机器挖矿，只希望保存数据，则修改config.json中的parallel_mine值为０
# 如果不想关注日志，可编辑 config.json 把 log_level 改为1
# 自己的私钥信息存在本目录下的.pKey文件中，需要做备份的可用keygen进行转换成.sKey存到其他地方
```

## 示例二: 上传证据到网络中

```shell

# 将client需要的config.json拷贝到当前目录，配置中指定的密钥也需要生成，操作可参考示例一


# 假设在my_evidence目录中有photo.jpg, description.txt, vedio.mp4三个文件，并希望统一对他们进行上链
./client -e ./my_evidence

# 命令行输出以下内容表示生成了摘要信息，并存到了　hf-my_evidence-20190824175510　文件中
# >>> generate hash file:/home/zhu/wp/996.Blockchain/bin/hf-my_evidence-20190824175510
# 
# 打开 hf-my_evidence-20190824175510 会看到类似内容
# {
#   "name": "my_evidence",
#   "hash": "16AF00C7DDE3C237425A40BEB49E2F01CBF90D0517A18FB9A78F9DC135470CF3",
#   "dir": [
#     {
#       "name": "description.txt",
#       "hash": "17E682F060B5F8E47EA04C5C4855908B0A5AD612022260FE50E11ECB0CC0AB76",
#       "dir": null
#     },
#     {
#       "name": "photo.jpg",
#       "hash": "3CF9A1A81F6BDEAF08A343C1E1C73E89CF44C06AC2427A892382CAE825E7C9C1",
#       "dir": null
#     },
#     {
#       "name": "vedio.mp4",
#       "hash": "5695D82A086B677962A0B0428ED1A213208285B7B40D7D3604876D36A710302A",
#       "dir": null
#     }
#   ]
# }
#
# 上面表示 my_evidence 这个目录的总摘要是16AF..，description.txt的摘要是17E6..等
# 总摘要可以根据其他摘要推理得出，上传证据时只会上传总的摘要信息

# 上传到链上
./client -u ./hf-my_evidence-20190824175510 -m "这里可以输入文字描述，该信息存在链上，明文可见"

# 此时需要等待几秒，上传成功后会有内容输出；此后再等待一段时间(约两个区块时间)，直到在链上查到自己的哈希信息，则说明上传成功了

# 查询刚才上传的哈希
./client -qe 16AF00C7DDE3C237425A40BEB49E2F01CBF90D0517A18FB9A78F9DC135470CF3
# 得到输出:
# Evidence <16AF00C7DDE3C237425A40BEB49E2F01CBF90D0517A18FB9A78F9DC135470CF3>
# [Version] 1
# [PubKey] 023352847EF717020F69AC07ADFC869FB4430FFC2EA525EFE9DF0ABD7583BF1AF6
# ......

# 查询本账户上传的证据
./client -qa
# 得到输出:
# Account	<AIZVFBD664LQED3JVQD237EGT62EGD74F2SSL37J34FL25MDX4NPM>
# Score:	0
# Evidence:
	1.16AF00C7DDE3C237425A40BEB49E2F01CBF90D0517A18FB9A78F9DC135470CF3
Finished.

# -qe的结果可以看到所在块的高度(比如100)，如果想看高度为100的块的信息，则
./client -qb 100

# 证据上传后，应该妥善保管上传时所用的密钥(即client配置文件中指定的密钥)，以及原始数据(my_evidence目录及里面的一切)
```

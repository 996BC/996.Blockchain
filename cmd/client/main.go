package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/utils"
)

func main() {
	configFile := flag.String("c", "./config.json", "the client config file")
	e := flag.String("e", "", "hash the evidence file or files under the directory recursively.")
	sha256Target := flag.String("sha256", "", "Use sha256 to hash the file and print the result")
	v := flag.String("v", "", "vefiry the hash file")
	u := flag.String("u", "", "upload the specified hash file content to the chain(only uploads the root hash)")
	m := flag.String("m", "", "add evidence description to the uploading hash file;it should be shorter than 140 characters, utf-8 encoding")
	qa := flag.Bool("qa", false, "query this account information")
	qe := flag.String("qe", "", `query the evidence information, you can seperate multiple parameters with ","`)
	qb := flag.String("qb", "", `query the specified height blocks information, 
support range format like "1-100", or multiple height seperated with ",", or the latest block with -1`)
	flag.Parse()

	var err error
	var conf *config
	var client *httpClient
	if len(*configFile) == 0 {
		fmt.Println("not found config file")
		return
	}
	if conf, err = parseConfig(*configFile); err != nil {
		fmt.Println(err)
		return
	}
	if client, err = initHTTPClient(conf); err != nil {
		fmt.Println(err)
		return
	}

	if len(*e) != 0 {
		err = generateHashFile(*e, conf.IgnoreHidden == 1)
	} else if len(*sha256Target) != 0 {
		_, err = getSha256HashOfFile(*sha256Target)
	} else if len(*v) != 0 {
		err = verifyHashFile(*v)
	} else if len(*u) != 0 {
		err = client.uploadHashFile(*u, *m)
	} else if *qa {
		err = client.queryAccount()
	} else if len(*qe) != 0 {
		err = client.queryEvidence(*qe)
	} else if len(*qb) != 0 {
		err = client.queryBlocks(*qb)
	} else {
		fmt.Printf("unknown operation")
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Filed:%v.\n", err)
		os.Exit(1)
	}
	fmt.Println("Finished.")
}

func initHTTPClient(conf *config) (*httpClient, error) {
	var privKey *btcec.PrivateKey
	var err error

	if conf.Key.Type == crypto.SealKeyType {
		privKey, err = crypto.RestoreSKey(conf.Key.Path)
	} else {
		privKey, err = crypto.RestorePKey(conf.Key.Path)
	}
	if err != nil {
		return nil, fmt.Errorf("restore key failed:%v", err)
	}

	var target uint64
	if target, err = strconv.ParseUint(conf.Difficulty, 16, 32); err != nil {
		return nil, fmt.Errorf("parse hash_diff failed:%v", err)
	}
	difficulty := blockchain.TargetToDiff(uint32(target))

	return newHTTPClient(conf.ServerIP,
		conf.ServerPort, conf.Scheme, privKey, difficulty), nil
}

func getSha256HashOfFile(file string) ([]byte, error) {
	if err := utils.AccessCheck(file); err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		return nil, fmt.Errorf("stat failed:%v", err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%s should be a file instead of a directory", file)
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read file failed:%v", err)
	}

	sum := utils.Hash(content)
	fmt.Printf("Sha256 of %s is %s\n", file, utils.ToHex(sum))
	return sum, nil
}

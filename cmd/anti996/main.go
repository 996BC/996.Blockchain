package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"net/http"
	_ "net/http/pprof"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/core"
	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/db"
	"github.com/996BC/996.Blockchain/p2p"
	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/rpc"
	"github.com/996BC/996.Blockchain/utils"
)

func main() {
	// load the config file
	cf := flag.String("c", "", "config file")
	pprofPort := flag.Int("pprof", 0, "pprof port, used by developers")
	flag.Parse()

	conf, err := parseConfig(*cf)
	if err != nil {
		log.Fatal(err)
	}
	utils.SetLogLevel(conf.LogLevel)
	logger := utils.GetStdoutLog()

	// load the key
	var privKey *btcec.PrivateKey
	if conf.Key.Type == crypto.PlainKeyType {
		privKey, err = crypto.RestorePKey(conf.Key.Path)
		if err != nil {
			logger.Fatal("restore sKey failed:%v\n", err)
		}
	}
	if conf.Key.Type == crypto.SealKeyType {
		privKey, err = crypto.RestoreSKey(conf.Key.Path)
		if err != nil {
			logger.Fatal("resotre pKey failed:%v\n", err)
		}
	}
	pubKey := privKey.PubKey()

	// p2p peer provider
	provider := peer.NewProvider(conf.IP, conf.Port, pubKey)
	seeds := parseSeeds(conf.Seeds)
	provider.AddSeeds(seeds)
	provider.Start()

	// p2p node
	nodeConfig := &p2p.Config{
		NodeIP:     conf.IP,
		NodePort:   conf.Port,
		Provider:   provider,
		MaxPeerNum: conf.MaxPeers,
		PrivKey:    privKey,
		Type:       conf.NodeType,
		ChainID:    conf.ChainID,
	}
	node := p2p.NewNode(nodeConfig)
	node.Start()

	// db
	if err = db.Init(conf.DataPath); err != nil {
		logger.Fatal("init db failed:%v\n", err)
	}
	logger.Info("database initialize successfully under the data path:%s\n", conf.DataPath)

	// core module
	blockDiffLimit, err := strconv.ParseUint(conf.BlockDifficultyLimit, 16, 32)
	if err != nil {
		logger.Fatalln(err)
	}
	evidenceDiffLimit, err := strconv.ParseUint(conf.EvidenceDifficultyLimit, 16, 32)
	if err != nil {
		logger.Fatalln(err)
	}
	coreInstance := core.NewCore(&core.Config{
		Node:         node,
		NodeType:     conf.NodeType,
		PrivKey:      privKey,
		ParallelMine: conf.ParallelMine,

		Config: &blockchain.Config{
			BlockTargetLimit:    uint32(blockDiffLimit),
			EvidenceTargetLimit: uint32(evidenceDiffLimit),
			BlockInterval:       conf.BlockInterval,
			Genesis:             conf.Genesis,
		},
	})

	// local http server
	httpConfig := &rpc.Config{
		Port: conf.HTTPPort,
		C:    coreInstance,
	}
	httpServer := rpc.NewServer(httpConfig)
	httpServer.Start()

	//pprof
	if *pprofPort != 0 {
		go func() {
			pprofAddress := fmt.Sprintf("localhost:%d", *pprofPort)
			log.Println(http.ListenAndServe(pprofAddress, nil))
		}()
	}

	// waiting gracefully shutdown
	sc := make(chan os.Signal)
	signal.Notify(sc, os.Interrupt)
	signal.Notify(sc, syscall.SIGTERM)
	select {
	case <-sc:
		logger.Infoln("Quiting......")
		httpServer.Stop()
		coreInstance.Stop()
		node.Stop()
		db.Close()
		logger.Infoln("Bye!")
		return
	}
}

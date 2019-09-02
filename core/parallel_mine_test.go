package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

func init() {
	// go test -v to check the goroutine exit log
	utils.SetLogLevel(utils.LogDebugLevel)
}

func TestParallelMine(t *testing.T) {
	pm := newParallelMine(2)
	header := cp.GenBlockHeaderFromParams(cp.NewBlockHeaderParams())
	difficulty := blockchain.TargetToDiff(0xEE100000)

	blockchain.BlockInterval = 1 * time.Hour
	job := pm.mine(difficulty, header)
	result := <-job.result

	if !result.found {
		t.Fatal("expect found nonce")
	}

	header.SetNonce(result.nonce)
	headerPow := header.GetPow()
	if headerPow.Cmp(difficulty) >= 0 {
		t.Fatal("error nonce")
	}

	// wait goroutine exit print
	time.Sleep(1 * time.Second)
}

func TestTimeout(t *testing.T) {
	pm := newParallelMine(2)
	header := cp.GenBlockHeaderFromParams(cp.NewBlockHeaderParams())
	difficulty := big.NewInt(0)

	blockchain.BlockInterval = 1 * time.Second

	begin := time.Now()
	job := pm.mine(difficulty, header)
	result := <-job.result
	end := time.Now()

	if result.found {
		t.Fatal("expect not found nonce")
	}

	if end.Sub(begin) > 3*time.Second {
		t.Fatal("expect timeout in 1.1 seconds")
	}

	// wait goroutine exit print
	time.Sleep(1 * time.Second)
}

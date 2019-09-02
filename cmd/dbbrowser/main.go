package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/db"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

var output *os.File

func main() {
	dbpath := flag.String("dbpath", "", `path of database`)
	r := flag.String("range", "", `view block via height range, like "1-100", "56", "-1"`)
	b := flag.String("b", "", `view block via hash in hex format`)
	e := flag.String("e", "", `view evidence via hash in hex format`)

	o := flag.String("o", "", `result output file; if it's null it will print to stdout`)
	flag.Parse()
	var err error

	if len(*dbpath) == 0 {
		fmt.Println("empty db path")
		os.Exit(1)
	}

	if err = utils.AccessCheck(*dbpath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = db.Init(*dbpath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(*o) != 0 {
		if err = utils.AccessCheck(*o); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		output, err = os.OpenFile(*o, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			fmt.Printf("open file %s failed:%v\n", *o, err)
			os.Exit(1)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	if len(*r) != 0 {
		err = rangeView(*r)
	} else if len(*b) != 0 {
		err = blockView(*b)
	} else if len(*e) != 0 {
		err = evidenceView(*e)
	} else {
		fmt.Println(`please input "-range" or "-b" or "-e" to choose what to view`)
		os.Exit(1)
	}
	if err != nil {
		fmt.Printf("error happen: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Finish.")
}

func rangeView(r string) error {
	var begin, end uint64

	num, err := strconv.ParseInt(r, 10, 64)
	if err == nil {
		if num == -1 {
			height, err := db.GetLatestHeight()
			if err != nil {
				return fmt.Errorf("err %v", err)
			}
			begin = height
			end = height
		} else if num > 0 {
			begin = uint64(num)
			end = uint64(num)
		} else {
			return fmt.Errorf("invalid index %d", num)
		}
	} else {
		n, err := fmt.Sscanf(r, "%d-%d", &begin, &end)
		if err != nil || n != 2 || begin >= end {
			return fmt.Errorf("invalid range")
		}
	}

	for i := begin; i <= end; i++ {
		block, hash, err := db.GetBlockViaHeight(i)
		if err != nil {
			return fmt.Errorf("get height %d block failed", i)
		}
		formatOutputBlock(block, hash, i)
	}

	return nil
}

func blockView(hash string) error {
	decoded, err := utils.FromHex(hash)
	if err != nil {
		return fmt.Errorf("decode %s failed", hash)
	}
	if len(decoded) != utils.HashLength {
		return fmt.Errorf("invalid hash size %d", len(decoded))
	}

	block, height, err := db.GetBlockViaHash(decoded)
	if err != nil {
		return fmt.Errorf("get block via %s failed", hash)
	}

	formatOutputBlock(block, decoded, height)
	return nil
}

func evidenceView(hash string) error {
	decoded, err := utils.FromHex(hash)
	if err != nil {
		return fmt.Errorf("decode %s failed", hash)
	}
	if len(decoded) != utils.HashLength {
		return fmt.Errorf("invalid hash size %d", len(decoded))
	}

	evidence, _, err := db.GetEvidenceViaHash(decoded)
	if err != nil {
		return fmt.Errorf("get evidence via %s failed", hash)
	}
	formatOutputEvidence(evidence)
	return nil
}

func wirte(format string, v ...interface{}) {
	if _, err := output.Write([]byte(fmt.Sprintf(format, v...))); err != nil {
		fmt.Printf("output err:%v\n", err)
		os.Exit(1)
	}
}

func formatOutputBlock(block *cp.Block, hash []byte, height uint64) {
	format :=
		`>>>>> [Block %d] %X
version		%d
time		%s
nonce		%d
difficulty	%X
lashHash	%X
miner		%X
evRoot		%s

`
	difficulty := blockchain.TargetToDiff(block.Target)

	var evRoot string
	empty := false
	if block.IsEmptyEvidenceRoot() {
		evRoot = "EMPTY"
		empty = true
	} else {
		evRoot = utils.ToHex(block.EvidenceRoot)
	}

	wirte(format, height, hash,
		block.Version,
		utils.TimeToString(block.Time),
		block.Nonce,
		difficulty,
		block.LastHash,
		block.Miner,
		evRoot)

	if empty {
		return
	}
	for _, evd := range block.Evds {
		formatOutputEvidence(evd)
	}
}

func formatOutputEvidence(evd *cp.Evidence) {
	format :=
		`[Evidence] %X
version			%d
owner_key		%X
signature		%X
description		%s
nonce			%d

`
	wirte(format, evd.Hash,
		evd.Version,
		evd.PubKey,
		evd.Sig,
		string(evd.Description),
		evd.Nonce)
}

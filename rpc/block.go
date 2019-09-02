package rpc

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/996BC/996.Blockchain/core"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	blockPath = "/block"
)

var (
	// BlocksV1Path /v1/block
	BlocksV1Path = version1Path + blockPath

	// QueryBlockViaRangeV1Path GET /v1/block/query-via-range
	QueryBlockViaRangeV1Path = BlocksV1Path + "/query-via-range"

	// QueryBlockViaHashV1Path GET /v1/block/query-via-hash
	QueryBlockViaHashV1Path = BlocksV1Path + "/query-via-hash"

	blockHandler = HTTPHandlers{
		{QueryBlockViaRangeV1Path, getBlockViaRange},
		{QueryBlockViaHashV1Path, getBlockViaHash},
	}
)

type GetBlocksResponse struct {
	Data []*BlockJSON `json:"data"`
}

type EvidenceInBlockJSON struct {
	Hash  string `json:"hash"`
	Owner string `json:"owner"`
}

type BlockJSON struct {
	Version      uint8                  `json:"version"`
	Time         int64                  `json:"time"`
	Nonce        uint32                 `json:"nonce"`
	Target       uint32                 `json:"target"`
	LastHash     string                 `json:"last_hash"`
	Miner        string                 `json:"miner"`
	EvidenceRoot string                 `json:"evidence_root"`
	Height       uint64                 `json:"height"`
	Hash         string                 `json:"hash"`
	Evds         []*EvidenceInBlockJSON `json:"evds`
}

func (b *BlockJSON) fromBlockInfo(info *core.BlockInfo) {
	b.Version = info.Version
	b.Time = info.Time
	b.Nonce = info.Nonce
	b.Target = info.Target
	b.LastHash = utils.ToHex(info.LastHash)
	b.Miner = crypto.BytesToID(info.Miner)
	b.EvidenceRoot = utils.ToHex(info.EvidenceRoot)
	b.Height = info.Height
	b.Hash = utils.ToHex(info.BlockHash)

	for _, evd := range info.Block.Evds {
		evdJSON := &EvidenceInBlockJSON{}
		evdJSON.Hash = utils.ToHex(evd.Hash)
		evdJSON.Owner = crypto.BytesToID(evd.PubKey)
		b.Evds = append(b.Evds, evdJSON)
	}
}

func responseBlocks(w http.ResponseWriter, blocks []*core.BlockInfo) {
	if len(blocks) == 0 {
		failedResponse("not found", w)
		return
	}

	if len(blocks) == 1 && blocks[0] == nil {
		failedResponse("not found", w)
		return
	}

	resp := &GetBlocksResponse{}
	for _, info := range blocks {
		blockJSON := &BlockJSON{}
		blockJSON.fromBlockInfo(info)
		resp.Data = append(resp.Data, blockJSON)
	}

	successWithDataResponse(resp, w)
}

/*
GET /v1/block/query-via-range?range=...

three kinds of range format:
1. from 1 to 100: 1-100
2. the specified height: 128 or 1,50,200 (separate with ,)
3. the latest block: -1
*/

type getBlockViaRangeResponse = GetBlocksResponse

func getBlockViaRange(w http.ResponseWriter, r *http.Request) {
	param, ok := r.URL.Query()[GetRangeParam]
	if !ok {
		badRequestResponse(w)
		return
	}

	// ..?range=xx
	height, err := strconv.ParseInt(param[0], 10, 64)
	if err == nil {
		if height == -1 {
			result := globalSvr.c.QueryLatestBlock()
			responseBlocks(w, []*core.BlockInfo{result})
			return
		}

		if height <= 0 {
			badRequestResponse(w)
			return
		}

		result := globalSvr.c.QueryBlockViaHeights([]uint64{uint64(height)})
		responseBlocks(w, result)
		return
	}

	// ..?range=xx-yy
	if strings.Contains(param[0], "-") {
		var begin, end uint64
		n, err := fmt.Sscanf(param[0], "%d-%d", &begin, &end)
		if err != nil || n != 2 || begin >= end {
			badRequestResponse(w)
			return
		}

		result := globalSvr.c.QueryBlockViaRange(begin, end)
		responseBlocks(w, result)
		return
	}

	// ..?range=xx,yy,zz
	if strings.Contains(param[0], ",") {
		heightsStr := strings.Split(param[0], ",")
		var heights []uint64
		for _, str := range heightsStr {
			height, err := strconv.ParseUint(str, 10, 64)
			if err != nil || height == 0 {
				badRequestResponse(w)
				return
			}
			heights = append(heights, height)
		}

		result := globalSvr.c.QueryBlockViaHeights(heights)
		responseBlocks(w, result)
		return
	}

	badRequestResponse(w)
}

/*
GET /v1/block/query-via-hash?hash=...

format: xxx or xxx,xxx,xxx (seperate with ,)
*/

type getBlockViaHashResponse = GetBlocksResponse

func getBlockViaHash(w http.ResponseWriter, r *http.Request) {
	param, ok := r.URL.Query()[GetHashParam]
	if !ok {
		badRequestResponse(w)
		return
	}

	var queryHash []string
	if strings.Contains(param[0], ",") {
		queryHash = strings.Split(param[0], ",")
	} else {
		queryHash = []string{param[0]}
	}

	result := globalSvr.c.QueryBlockViaHash(queryHash)
	responseBlocks(w, result)
	return
}

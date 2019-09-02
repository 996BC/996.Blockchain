package rpc

import (
	"net/http"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	accountPath = "/account"
)

var (
	// AccountV1Path /v1/account
	AccountV1Path = version1Path + accountPath

	// QueryAccountV1Path GET /v1/account
	QueryAccountV1Path = AccountV1Path + "/query"

	accountHandlers = HTTPHandlers{
		{QueryAccountV1Path, getAccount},
	}
)

/*
GET /v1/account/query?id=...
*/
type GetAccountResponse struct {
	Evidence []string `json:"evidence"`
	Score    uint64   `json:"score"`
}

func getAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := r.URL.Query()[GetIDParam]
	if !ok {
		badRequestResponse(w)
		return
	}

	key := crypto.IDToBytes(id[0])
	if key == nil || len(key) != btcec.PubKeyBytesLenCompressed {
		badRequestResponse(w)
		return
	}

	evdsHash, score := globalSvr.c.QueryAccount(id[0])
	if evdsHash == nil && score == 0 {
		failedResponse("Not found account", w)
		return
	}

	var evidence []string
	for _, h := range evdsHash {
		evidence = append(evidence, utils.ToHex(h))
	}

	successWithDataResponse(&GetAccountResponse{
		Evidence: evidence,
		Score:    score,
	}, w)
}

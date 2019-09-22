package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/996BC/996.Blockchain/core"
	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

const (
	evdPath          = "/evidence"
	maxBatchQueryNum = 40
)

var (

	// EvidenceV1Path /v1/evidence
	EvidenceV1Path = version1Path + evdPath

	// UploadEvidenceV1Path POST /v1/evidence/upload
	UploadEvidenceV1Path = EvidenceV1Path + "/upload"

	// UploadEvidenceRawV1Path POST /v1/evidence/upload-raw
	UploadEvidenceRawV1Path = version1Path + evdPath + "/upload-raw"

	// QueryEvidenceV1Path POST /v1/evidence/query
	QueryEvidenceV1Path = EvidenceV1Path + "/query"

	evidenceHandlers = HTTPHandlers{
		{UploadEvidenceV1Path, uploadEvds},
		{UploadEvidenceRawV1Path, uploadRaw},
		{QueryEvidenceV1Path, queryEvidence},
	}
)

type EvidenceJSON struct {
	Version     uint8  `json:"version"`
	Hash        string `json:"hash"`
	Description string `json:"description"`
	PubKey      string `json:"public_key"`
	Sig         string `json:"sigature"`
	Nonce       uint32 `json:"nonce"`
	Height      uint64 `json:"height"`
	BlockHash   string `json:"block_hash"`
	Time        int64  `json:"time"`
}

func (e *EvidenceJSON) toCPEvidence() *cp.Evidence {
	if e.Version != cp.CoreProtocolV1 {
		return nil
	}
	var hash []byte
	var pubKey []byte
	var sig []byte
	var err error

	if hash, err = utils.FromHex(e.Hash); err != nil {
		return nil
	}
	if err := cp.VerifyDescription(e.Description); err != nil {
		return nil
	}
	if pubKey, err = utils.FromHex(e.PubKey); err != nil {
		return nil
	}
	if sig, err = utils.FromHex(e.Sig); err != nil {
		return nil
	}

	result := cp.NewEvidenceV1(hash, []byte(e.Description), pubKey)
	result.SetNonce(e.Nonce)
	result.Sig = sig

	return result
}

func (e *EvidenceJSON) fromEvidenceInfo(info *core.EvidenceInfo) {
	e.Version = info.Version
	e.Hash = utils.ToHex(info.Hash)
	e.Description = string(info.Description)
	e.PubKey = utils.ToHex(info.PubKey)
	e.Sig = utils.ToHex(info.Sig)
	e.Nonce = info.Nonce
	e.Height = info.Height
	e.BlockHash = utils.ToHex(info.BlockHash)
	e.Time = info.Time
}

/*
POST /v1/evidence/upload
{
	"data": [{
		"version": 1,
		"hash": "xxxx",
		"description": "xxxx",
		"public_key": "xxxx",
		"sigature": "xxxx",
		"nonce": 10000
	}]
}
*/
type UploadEvdsReq struct {
	Data []*EvidenceJSON `json:"data"`
}

func uploadEvds(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		badRequestResponse(w)
		return
	}

	query := &UploadEvdsReq{}
	if err := json.Unmarshal(body, query); err != nil {
		badRequestResponse(w)
		return
	}

	var evds []*cp.Evidence
	for _, data := range query.Data {
		evd := data.toCPEvidence()
		if evd == nil {
			badRequestResponse(w)
			return
		}
		evds = append(evds, evd)
	}

	if err := globalSvr.c.UploadEvidence(evds); err != nil {
		if existErr, ok := err.(blockchain.ErrEvidenceAlreadyExist); ok {
			failedResponse(existErr.Error(), w)
			return
		}

		logger.Info("upload failed:%v\n", err)
		badRequestResponse(w)
		return
	}

	successResponse(w)
}

/*
POST /v1/evidence/upload-raw
{
   "evds":[
      {
         "hash":"xxx",
         "description":"yyy"
      },
      {
         "hash":"xxx",
         "description":"yyy"
      }
   ]
}
*/
type rawEvidenceJSON struct {
	Hash        string `json:"hash"`
	Description string `json:"description"`
}

func (r *rawEvidenceJSON) toRawEvidence() (*core.RawEvidence, error) {
	result := &core.RawEvidence{}
	var err error

	if result.Hash, err = utils.FromHex(r.Hash); err != nil {
		return nil, fmt.Errorf("invalid hash:%v", err)
	}
	if err = cp.VerifyDescription(r.Description); err != nil {
		return nil, err
	}
	result.Description = []byte(r.Description)

	return result, nil
}

type uploadRawReq struct {
	evds []*rawEvidenceJSON `json:"evds"`
}

func uploadRaw(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		badRequestResponse(w)
		return
	}

	query := &uploadRawReq{}
	if err := json.Unmarshal(body, query); err != nil {
		badRequestResponse(w)
		return
	}

	var rEvds []*core.RawEvidence
	for _, evd := range query.evds {
		rEvd, err := evd.toRawEvidence()
		if err != nil {
			badRequestResponse(w)
			return
		}

		rEvds = append(rEvds, rEvd)
	}

	if err := globalSvr.c.UploadEvidenceRaw(rEvds); err != nil {
		logger.Info("upload raw failed:%v\n", err)
		badRequestResponse(w)
		return
	}

	successResponse(w)
}

/*
POST /v1/evidence/query
{
	"hash":["xxxx", "xxxx", "xxxx"]
}
*/

type QueryEvidenceReq struct {
	Hash []string `json:"hash"`
}

type QueryEvidenceResp struct {
	Data []*EvidenceJSON `json:"data"`
}

func queryEvidence(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		badRequestResponse(w)
		return
	}

	query := &QueryEvidenceReq{}
	if err := json.Unmarshal(body, query); err != nil {
		badRequestResponse(w)
		return
	}

	if len(query.Hash) == 0 || len(query.Hash) > maxBatchQueryNum {
		badRequestResponse(w)
		return
	}

	for _, hexHash := range query.Hash {
		h, err := utils.FromHex(hexHash)
		if err != nil || len(h) != utils.HashLength {
			badRequestResponse(w)
			return
		}
	}

	evidenceInfo := globalSvr.c.QueryEvidence(query.Hash)
	if evidenceInfo == nil {
		failedResponse("Not found evidence", w)
		return
	}

	resp := &QueryEvidenceResp{}
	for _, e := range evidenceInfo {
		eJSON := &EvidenceJSON{}
		eJSON.fromEvidenceInfo(e)
		resp.Data = append(resp.Data, eJSON)
	}

	successWithDataResponse(resp, w)
	return
}

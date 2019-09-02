package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/996BC/996.Blockchain/core/blockchain"
	"github.com/996BC/996.Blockchain/crypto"
	"github.com/996BC/996.Blockchain/rpc"
	"github.com/996BC/996.Blockchain/serialize/cp"
	"github.com/996BC/996.Blockchain/utils"
)

type httpClient struct {
	serverIP   string
	serverPort string
	scheme     string
	privKey    *btcec.PrivateKey
	client     *http.Client
	Difficulty *big.Int
}

func newHTTPClient(ip string, port int, scheme string,
	key *btcec.PrivateKey, difficulty *big.Int) *httpClient {
	return &httpClient{
		serverIP:   ip,
		serverPort: strconv.Itoa(port),
		scheme:     scheme,
		privKey:    key,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Difficulty: difficulty,
	}
}

func (hc *httpClient) uploadHashFile(file, description string) error {
	if err := utils.AccessCheck(file); err != nil {
		return err
	}

	jsonBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read hash file failed:%v", err)
	}

	hf := &hashFile{}
	if err := json.Unmarshal(jsonBytes, hf); err != nil {
		return fmt.Errorf("parse hash file failed:%v", err)
	}

	if len(hf.Name) == 0 || len(hf.Hash) == 0 {
		return fmt.Errorf("invalid hash file")
	}

	evd, err := hc.generateEvidence(hf.Hash, description)
	if err != nil {
		return err
	}

	fileOrDir := "file"
	if hf.Dir != nil {
		fileOrDir = "directory"
	}
	fmt.Printf("Ready to upload hash %s (of %s %s)...\n", hf.Hash, fileOrDir, hf.Name)
	return hc.uploadEvidence(evd)
}

func (hc *httpClient) generateEvidence(hash, description string) (*cp.Evidence, error) {
	h, err := utils.FromHex(hash)
	if err != nil {
		return nil, fmt.Errorf("hex decode hash failed:%v", err)
	}
	if len(h) != utils.HashLength {
		return nil, fmt.Errorf("invalid hash length")
	}

	if err := cp.VerifyDescription(description); err != nil {
		return nil, err
	}

	pubKeyB := hc.privKey.PubKey().SerializeCompressed()
	evd := cp.NewEvidenceV1(h, []byte(description), pubKeyB)
	evd.Sign(hc.privKey)

	fmt.Printf("doing pow for your evidence, wait...\n")
	pow := evd.NextNonce()
	for pow.Cmp(hc.Difficulty) >= 0 {
		pow = evd.NextNonce()
	}

	return evd, nil
}

func (hc *httpClient) uploadEvidence(evd *cp.Evidence) error {
	evdJSON := &rpc.EvidenceJSON{
		Version:     cp.CoreProtocolV1,
		Hash:        utils.ToHex(evd.Hash),
		Description: string(evd.Description),
		PubKey:      utils.ToHex(evd.PubKey),
		Sig:         utils.ToHex(evd.Sig),
		Nonce:       evd.Nonce,
		// ignore other fileds
	}

	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse
	var requestBody []byte

	if requestBody, err = json.Marshal(
		&rpc.UploadEvdsReq{
			Data: []*rpc.EvidenceJSON{evdJSON}}); err != nil {
		return err
	}

	if req, err = hc.genRequest(http.MethodPost, rpc.UploadEvidenceV1Path, nil, nil, requestBody); err != nil {
		return err
	}

	if httpResp, err = hc.client.Do(req); err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	if rpcResp, err = hc.parseResponse(httpResp, nil); err != nil {
		return err
	}

	handler := func() {
		fmt.Println(">>> upload hash successfully")
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) queryAccount() error {
	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse

	accountID := crypto.PrivKeyToID(hc.privKey)
	if req, err = hc.genRequest(http.MethodGet, rpc.QueryAccountV1Path,
		[]string{rpc.GetIDParam}, []string{accountID}, nil); err != nil {
		return err
	}

	if httpResp, err = hc.client.Do(req); err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	accnountJSON := &rpc.GetAccountResponse{}
	if rpcResp, err = hc.parseResponse(httpResp, accnountJSON); err != nil {
		return err
	}

	handler := func() {
		content := "Account\t<%s>\nScore:\t%d\nEvidence:\n%s\n"

		var evidenceContent string
		for i := 0; i < len(accnountJSON.Evidence); i++ {
			record := fmt.Sprintf("\t%d.%s\n", i+1, accnountJSON.Evidence[i])
			evidenceContent += record
		}

		fmt.Printf(content, accountID, accnountJSON.Score, evidenceContent)
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) queryEvidence(params string) error {
	hexHashs := strings.Split(params, ",")
	for _, hash := range hexHashs {
		h, err := utils.FromHex(hash)
		if err != nil || len(h) != utils.HashLength {
			return fmt.Errorf("invalid evidence hash %s", hash)
		}
	}

	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse
	var requestBody []byte

	if requestBody, err = json.Marshal(&rpc.QueryEvidenceReq{Hash: hexHashs}); err != nil {
		return err
	}

	if req, err = hc.genRequest(http.MethodPost, rpc.QueryEvidenceV1Path, nil, nil, requestBody); err != nil {
		return err
	}

	httpResp, err = hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	queryEvidenceResp := &rpc.QueryEvidenceResp{}
	if rpcResp, err = hc.parseResponse(httpResp, queryEvidenceResp); err != nil {
		return err
	}

	handler := func() {
		for _, evd := range queryEvidenceResp.Data {
			content := "Evidence <%s>\n[Version] %d\n[PubKey] %s\n[Signature] %s\n[Description] %s\n[Nonce] %d\n[Height] %d\n[Block] %s\n[Time] %s\n\n"

			fmt.Println("--------------------------------------------------------")
			fmt.Printf(content, evd.Hash, evd.Version, evd.PubKey, evd.Sig, evd.Description,
				evd.Nonce, evd.Height, evd.BlockHash, utils.TimeToString(evd.Time))
		}
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) queryBlocks(params string) error {
	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse

	if req, err = hc.genRequest(http.MethodGet, rpc.QueryBlockViaRangeV1Path,
		[]string{rpc.GetRangeParam}, []string{params}, nil); err != nil {
		return err
	}

	if httpResp, err = hc.client.Do(req); err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	blocksResponse := &rpc.GetBlocksResponse{}
	if rpcResp, err = hc.parseResponse(httpResp, blocksResponse); err != nil {
		return err
	}

	handler := func() {
		for _, block := range blocksResponse.Data {
			blockContent := `
Block <%s> Height:%d
Time		%s
Version		%d
Nonce		%d
Difficulty	%X
LastBlock	%s
Miner		%s
Root		%s

Evidece details:
No	Hash									Owner
%s
`

			var evidenceContent string
			for i := 0; i < len(block.Evds); i++ {
				evidenceContent += fmt.Sprintf("[%d]\t%s\t%s\n", i,
					block.Evds[i].Hash, block.Evds[i].Owner)
			}

			diff := blockchain.TargetToDiff(block.Target)
			fmt.Printf(blockContent, block.Hash, block.Height,
				utils.TimeToString(block.Time),
				block.Version,
				block.Nonce,
				diff,
				block.LastHash,
				block.Miner,
				block.EvidenceRoot,
				evidenceContent,
			)

			fmt.Println("--------------------------------------------------------")
		}
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) genRequest(method string, path string, key, value []string, postData []byte) (*http.Request, error) {
	u, _ := url.Parse(hc.scheme + "://" + hc.serverIP + ":" + hc.serverPort)
	u.Path = path

	q := u.Query()
	for i := 0; i < len(key); i++ {
		q.Add(key[i], value[i])
	}
	u.RawQuery = q.Encode()

	var httpBody io.Reader
	if postData != nil {
		httpBody = bytes.NewBuffer(postData)
	}

	req, err := http.NewRequest(method, u.String(), httpBody)
	if err != nil {
		return nil, fmt.Errorf("generate query failed:%v", err)
	}

	return req, nil
}

func (hc *httpClient) parseResponse(resp *http.Response, data interface{}) (*rpc.HTTPResponse, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed, return:%d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read http body failed:%v", err)
	}

	httpResponse := rpc.ParseHTTPResponse(bodyBytes, data)
	if httpResponse == nil {
		return nil, fmt.Errorf("unmarshal response json failed")
	}

	return httpResponse, nil
}

func (hc *httpClient) responseHandle(httpResponse *rpc.HTTPResponse, f func()) {
	switch httpResponse.Code {
	case rpc.CodeSuccess:
		f()
	case rpc.CodeFailed:
		fmt.Printf("failed: %s\n", httpResponse.Message)
	case rpc.CodeBadRequest:
		fmt.Println("bad request, please check your input")
	default:
		fmt.Printf("response unknown code:%d\n", httpResponse.Code)
	}
}

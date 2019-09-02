package rpc

import (
	"encoding/json"
	"net/http"
)

const (
	CodeSuccess    = 0
	CodeFailed     = 1
	CodeBadRequest = 2
)

type HTTPResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func ParseHTTPResponse(j []byte, expectData interface{}) *HTTPResponse {
	result := &HTTPResponse{Data: expectData}
	if err := json.Unmarshal(j, result); err != nil {
		return nil
	}
	return result
}

func doResponse(code int, msg string, data interface{}, w http.ResponseWriter) {
	resp := &HTTPResponse{
		Code:    code,
		Message: msg,
		Data:    data,
	}

	respB, err := json.Marshal(resp)
	if err != nil {
		logger.Warn("json marshal HTTPResponse failed:%v\n", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respB)
}

func successResponse(w http.ResponseWriter) {
	doResponse(CodeSuccess, "", nil, w)
}

func successWithDataResponse(data interface{}, w http.ResponseWriter) {
	doResponse(CodeSuccess, "", data, w)
}

func failedResponse(msg string, w http.ResponseWriter) {
	doResponse(CodeFailed, msg, nil, w)
}

func badRequestResponse(w http.ResponseWriter) {
	doResponse(CodeBadRequest, "", nil, w)
}

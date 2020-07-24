package jsonrpc

import "net/http"

type RequestBodyBase struct {
	Version string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Method  string `json:"method"`
}

type RequestBodyParams struct {
	Params interface{} `json:"params,omitempty"`
}

type RequestBody struct {
	RequestBodyBase
	RequestBodyParams
}

type Request struct {
	Params  interface{}
	Header  http.Header
	Cookies []*http.Cookie
}

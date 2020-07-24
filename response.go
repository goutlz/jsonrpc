package jsonrpc

import "net/http"

type ResponseBodyBase struct {
	Version string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Err     *Error `json:"error,omitempty"`
}

type ResponseBodyResult struct {
	Result interface{} `json:"result,omitempty"`
}

type ResponseBody struct {
	ResponseBodyBase
	ResponseBodyResult
}

type Response struct {
	Data    interface{}
	Header  http.Header
	Cookies []*http.Cookie
}

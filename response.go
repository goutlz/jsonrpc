package jsonrpc

type MethodResponseBase struct {
	RawID   *rawId       `json:"id"`
	Version string       `json:"jsonrpc"`
	Err     *ServerError `json:"error,omitempty"`
}

type MethodResponse struct {
	MethodResponseBase `json:",inline"`
	Result             interface{} `json:"result,omitempty"`
}

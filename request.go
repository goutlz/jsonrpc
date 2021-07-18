package jsonrpc

import (
	"bytes"
	"encoding/json"

	"github.com/goutlz/errz"
)

type MethodRequestBase struct {
	RawID          *rawId `json:"id,omitempty"`
	JsonRpcVersion string `json:"jsonrpc"`
	MethodName     string `json:"method"`
}

func (b *MethodRequestBase) ID() string {
	return b.RawID.StringValue
}

type MethodRequest struct {
	MethodRequestBase `json:",inline"`
	Params            interface{} `json:"params,omitempty"`
}

type RequestParams interface {
	Validate() error
}

type MethodRequestRaw interface{}

func rawToMethodRequest(raw MethodRequestRaw) *MethodRequest {
	bytesRawReq, err := json.Marshal(raw)
	if err != nil {
		err = errz.Wrap(err, "Failed to marshal raw request")
		makeErrorAndThrow(InternalErrorCode, err)
	}

	var req MethodRequest
	err = json.NewDecoder(bytes.NewReader(bytesRawReq)).Decode(&req)
	if err != nil {
		err = errz.Wrapf(err, "Failed to decode params to %T", req)
		makeErrorAndThrow(InvalidRequestCode, err)
	}

	return &req
}

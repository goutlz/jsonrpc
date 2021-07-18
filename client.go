package jsonrpc

import (
	"bytes"
	"encoding/json"
	"github.com/goutlz/errz"
	"io/ioutil"
	"net/http"
)

type ClientRequestArgs struct {
	Method        string
	Params        interface{}
	ResultFactory func() interface{}
}

type Client interface {
	Call(args *ClientRequestArgs) (*MethodResponse, error)
	CallBatch(argsBatch []*ClientRequestArgs) ([]*MethodResponse, error)
}

type client struct {
	url           string
	httpClient    *http.Client
	rpcVersion    string
	generateRawId generateRawIdFunc
}

func NewClient(opts *ClientOpts) (Client, error) {
	return &client{
		url:           opts.Url,
		httpClient:    opts.HttpClient,
		rpcVersion:    version,
		generateRawId: newGenerateIdFunc(opts.getIdFactory()),
	}, nil
}

func (c *client) Call(args *ClientRequestArgs) (response *MethodResponse, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		response = nil
		err, ok := r.(error)
		if ok {
			err = errz.Wrap(err, "Failed to make a call. Panic occurred")
			return
		}
		err = errz.Newf("Failed to make a call. Panic occurred with unknown error: %+v. Type: %T.", err, err)
	}()

	request, err := c.makeMethodRequestObject(args)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to make method request object")
	}

	respBytes, err := c.doRequest(request)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to do request")
	}

	respToUnmarshal := MethodResponse{Result: args.ResultFactory()}
	err = json.Unmarshal(respBytes, &respToUnmarshal)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to unmarshal response")
	}

	return &respToUnmarshal, nil
}

func (c *client) CallBatch(argsBatch []*ClientRequestArgs) (responses []*MethodResponse, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		responses = nil
		err, ok := r.(error)
		if ok {
			err = errz.Wrap(err, "Failed to make a batch call. Panic occurred")
			return
		}
		err = errz.Newf("Failed to make a batch call. Panic occurred with unknown error: %+v. Type: %T.", err, err)
	}()

	var requests []*MethodRequest

	for _, args := range argsBatch {
		request, err := c.makeMethodRequestObject(args)
		if err != nil {
			return nil, errz.Wrap(err, "Failed to make method request object")
		}

		requests = append(requests, request)

		if args.ResultFactory == nil {
			continue
		}

		responses = append(responses, &MethodResponse{Result: args.ResultFactory()})
	}

	respBytes, err := c.doRequest(requests)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to do batch request")
	}

	var singleErrorResponse *MethodResponse
	err = json.Unmarshal(respBytes, &singleErrorResponse)
	if err == nil {
		return nil, errz.Wrapf(err, "Failed to do batch request: %+v", *singleErrorResponse)
	}

	err = json.Unmarshal(respBytes, &responses)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to unmarshal responses")
	}

	return responses, nil
}

func (c *client) makeMethodRequestObject(args *ClientRequestArgs) (*MethodRequest, error) {
	request := &MethodRequest{
		MethodRequestBase: MethodRequestBase{
			JsonRpcVersion: c.rpcVersion,
			MethodName:     args.Method,
		},
		Params: args.Params,
	}

	if args.ResultFactory != nil {
		generatedId, err := c.generateRawId()
		if err != nil {
			return nil, errz.Wrap(err, "Failed to generate rawId")
		}

		request.RawID = generatedId
	}

	return request, nil
}

func (c *client) doRequest(request interface{}) ([]byte, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to marshal request")
	}

	r := bytes.NewReader(data)
	req, err := http.NewRequest("POST", c.url, r)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to create http-request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, errz.Wrap(err, "Failed to send request")
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to read response")
	}

	return respData, nil
}

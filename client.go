package jsonrpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/goutlz/errz"

	"github.com/google/uuid"
)

type ResponseResultFactory func() interface{}
type IDFactory func() string

type Client interface {
	Call(url string, method string, args Request, expectedResult ResponseResultFactory) (*ResponseBody, error)
}

type client struct {
	httpClient *http.Client
	rpcVersion string
	getID      IDFactory
}

func (c *client) Call(url string, method string, args Request, expectedResult ResponseResultFactory) (responseBody *ResponseBody, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		responseBody = nil
		err, ok := r.(error)
		if ok {
			err = errz.Wrap(err, "Failed to make a call. Panic occurred")
			return
		}
		err = errz.Newf("Failed to make a call. Panic occurred with unknown error: %+v. Type: %T.", err, err)
	}()

	request := RequestBody{
		RequestBodyBase{
			Method:  method,
			ID:      c.getID(),
			Version: c.rpcVersion,
		},
		RequestBodyParams{
			Params: args.Params,
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to marshal request")
	}

	r := bytes.NewReader(data)
	req, err := http.NewRequest("POST", url, r)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to create http-request")
	}

	if args.Header != nil {
		req.Header = args.Header
	}
	req.Header.Set("Content-Type", "application/json")
	for i := range args.Cookies {
		req.AddCookie(args.Cookies[i])
	}

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

	respBody := ResponseBody{
		ResponseBodyResult: ResponseBodyResult{
			Result: expectedResult(),
		},
	}

	err = json.Unmarshal(respData, &respBody)
	if err != nil {
		return nil, errz.Wrap(err, "Failed to unmarshal response")
	}

	return &respBody, nil
}

func guidID() string {
	return uuid.New().String()
}

func NewClient(httpClient *http.Client) Client {
	return &client{
		httpClient: httpClient,
		rpcVersion: version,
		getID:      guidID,
	}
}

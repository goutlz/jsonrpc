package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	version = "2.0"
)

type HandlerInfo struct {
	Handler         RequestHandler
	ContextBuilders []ContextBuilder
	ParamsFactory   RequestParamsFactory
}

type ContextBuilder func(ctx context.Context, request *RequestBodyBase, rawHttpRequest *http.Request) (context.Context, ServerError)
type RequestParamsFactory func() interface{}
type RequestHandler func(ctx context.Context, request *Request) (*Response, ServerError)
type RouteHandler map[string]HandlerInfo

func createJsonRpcHandler(handler RouteHandler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer r.Body.Close()
		}

		var response []byte
		defer func() {
			rVal := recover()
			if rVal == nil {
				w.Write(response)
				return
			}

			errResponse := ResponseBodyBase{
				Version: version,
			}

			jsonRpcErr, ok := rVal.(ServerError)
			if !ok {
				errResponse.Err = &Error{
					Code:    MakeModuleErrorCode(jsonrpc_module_code, 0),
					Message: "Unknown error",
					Data:    rVal,
				}
			} else {
				errResponse.Err = jsonRpcErr.ToError()
			}

			response, err := json.Marshal(errResponse)
			if err == nil {
				w.Write(response)
				return
			}

			w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}()

		ctx := context.Background()

		handlerResponse, requestId := handleRequest(ctx, handler, r)
		applyCookies(w, handlerResponse.Cookies)
		applyHeaders(w.Header(), handlerResponse.Header)

		response, err := json.Marshal(ResponseBody{
			ResponseBodyBase: ResponseBodyBase{
				Version: version,
				ID:      requestId,
			},
			ResponseBodyResult: ResponseBodyResult{
				Result: handlerResponse.Data,
			},
		})

		if err != nil {
			makeErrorAndThrow(6, "Failed to marshal response", err)
		}
	}
}

func handleRequest(ctx context.Context, handler RouteHandler, r *http.Request) (*Response, string) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		makeErrorAndThrow(1, "Failed to read request body", err)
	}

	var bodyBase RequestBodyBase
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&bodyBase)
	if err != nil {
		makeErrorAndThrow(2, "Failed to parse request body", err)
	}

	if bodyBase.Version != version {
		makeErrorAndThrow(3, "Unsupported JSON RPC version", errors.New(fmt.Sprintf("expected version = %s", version)))
	}

	handlerInfo, ok := handler[bodyBase.Method]
	if !ok {
		makeErrorAndThrow(4, fmt.Sprintf("Unsupported method: %s", bodyBase.Method), nil)
	}

	bodyParams := RequestBodyParams{
		Params: handlerInfo.ParamsFactory(),
	}
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&bodyParams)
	if err != nil {
		makeErrorAndThrow(5, "Failed to parse params for handler", err)
	}

	var serverError ServerError
	for _, builder := range handlerInfo.ContextBuilders {
		ctx, serverError = builder(ctx, &bodyBase, r)
		if serverError != nil {
			throwError(serverError)
		}
	}

	request := &Request{
		Params:  bodyParams.Params,
		Header:  r.Header,
		Cookies: r.Cookies(),
	}

	handlerResp, serverErr := handlerInfo.Handler(ctx, request)
	if serverErr != nil {
		throwError(serverErr)
	}

	return handlerResp, bodyBase.ID
}

func applyCookies(w http.ResponseWriter, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		http.SetCookie(w, cookie)
	}
}

func applyHeaders(w http.Header, headers http.Header) {
	for k, hs := range headers {
		for i := range hs {
			w.Add(k, hs[i])
		}
	}
}

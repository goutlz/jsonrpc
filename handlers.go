package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync"

	"github.com/goutlz/errz"
)

const (
	version = "2.0"
)

type HandlerInfo struct {
	Handler       Handler
	ParamsFactory ParamsFactory
	ResultFactory ResultFactory
	Middlewares   []HandlerMiddleware
}

type MethodHandlers map[string]HandlerInfo
type HandlerMiddleware func(ctx context.Context, request *MethodRequest) (context.Context, *ServerError)
type ParamsFactory func() RequestParams
type Handler func(ctx context.Context, params RequestParams) (interface{}, *ServerError)
type ResultFactory func(ctx context.Context, result interface{}) interface{}

type jsonRpcHandler struct {
	handlePanicResponse func(interface{}) *MethodResponse
	methodHandlers      MethodHandlers
}

func createJsonRpcHandler(handlers MethodHandlers) http.Handler {
	return &jsonRpcHandler{
		handlePanicResponse: handlePanicResponse,
		methodHandlers:      handlers,
	}
}

func (j *jsonRpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}

	defer func() {
		panicResponse := j.handlePanicResponse(recover())
		if panicResponse == nil {
			return
		}

		response, err := json.Marshal(panicResponse)
		if err == nil {
			w.Write(response)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(internalErrorMessage))
	}()

	response := j.handleRequest(r)
	w.Write(response)
}

func (j *jsonRpcHandler) handleRequest(r *http.Request) []byte {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errz.Wrap(err, "Failed to read request body")
		makeErrorAndThrow(InternalErrorCode, err)
	}

	var batchRequestRaw []MethodRequestRaw
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&batchRequestRaw)
	if err == nil {
		return marshalResponse(j.handleBatch(batchRequestRaw))
	}

	var singleRequest MethodRequestRaw
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&singleRequest)
	if err == nil {
		return marshalResponse(j.handleSingle(singleRequest))
	}

	err = errz.Wrap(err, "Failed to parse request body")
	makeErrorAndThrow(ParseErrorCode, err)
	return nil
}

func (j *jsonRpcHandler) handleSingle(raw MethodRequestRaw) (response *MethodResponse) {
	defer func() {
		panicResponse := j.handlePanicResponse(recover())
		if panicResponse == nil {
			return
		}

		response = panicResponse
	}()

	req := rawToMethodRequest(raw)

	if req.JsonRpcVersion != version {
		err := errz.Newf("Unsupported JSON RPC version: %s. Expected: %s", req.JsonRpcVersion, version)
		makeErrorAndThrow(InvalidRequestCode, err)
	}

	handlerInfo, ok := j.methodHandlers[req.MethodName]
	if !ok {
		makeErrorAndThrow(MethodNotFoundCode, nil)
	}

	bytesParams, err := json.Marshal(req.Params)
	if err != nil {
		err = errz.Wrap(err, "Failed to marshal params")
		makeErrorAndThrow(InternalErrorCode, err)
	}

	var params RequestParams
	if handlerInfo.ParamsFactory != nil {
		params = handlerInfo.ParamsFactory()
		err = json.NewDecoder(bytes.NewReader(bytesParams)).Decode(&params)
		if err != nil {
			err = errz.Wrapf(err, "Failed to decode params to %T", params)
			makeErrorAndThrow(InvalidRequestCode, err)
		}

		err = params.Validate()
		if err != nil {
			err = errz.Wrap(err, "Failed to validate params")
			makeErrorAndThrow(InvalidParamsCode, err)
		}
	}

	ctx := context.Background()
	for _, middleware := range handlerInfo.Middlewares {
		req.Params = params
		middlewareCtx, serverError := middleware(ctx, req)
		if serverError != nil {
			throwError(serverError)
		}

		ctx = middlewareCtx
	}

	result, serverErr := handlerInfo.Handler(ctx, params)
	if serverErr != nil {
		throwError(serverErr)
	}

	if req.RawID == nil {
		return nil
	}

	if handlerInfo.ResultFactory != nil {
		result = handlerInfo.ResultFactory(ctx, result)
	}

	return &MethodResponse{
		MethodResponseBase: MethodResponseBase{
			Version: version,
			RawID:   req.RawID,
		},
		Result: result,
	}
}

func (j *jsonRpcHandler) handleBatch(requests []MethodRequestRaw) []*MethodResponse {
	reqCount := len(requests)
	if reqCount == 0 {
		makeErrorAndThrow(InvalidRequestCode, errz.New("Empty batch array"))
	}

	var wg sync.WaitGroup
	resp := make([]*MethodResponse, reqCount)

	for i, req := range requests {
		wg.Add(1)

		go func(index int, raw MethodRequestRaw) {
			defer wg.Done()

			singleResp := j.handleSingle(raw)
			if singleResp == nil {
				return
			}

			resp[index] = singleResp
		}(i, req)
	}

	wg.Wait()

	return filterEmptyResponses(resp)
}

func handlePanicResponse(recovered interface{}) *MethodResponse {
	if recovered == nil {
		return nil
	}

	errResponse := MethodResponse{
		MethodResponseBase: MethodResponseBase{
			Version: version,
		},
	}

	jsonRpcErr, ok := recovered.(*ServerError)
	if !ok {
		errResponse.Err = MakeBuiltInError(InternalErrorCode, errz.Newf("Recovered error: %v", recovered))
	} else {
		errResponse.Err = jsonRpcErr
	}

	return &errResponse
}

func marshalResponse(resp interface{}) []byte {
	if resp == nil || reflect.ValueOf(resp).IsNil() {
		return nil
	}

	bytesResp, err := json.Marshal(resp)
	if err != nil {
		err = errz.Wrap(err, "Failed to marshal response")
		makeErrorAndThrow(InternalErrorCode, err)
	}

	return bytesResp
}

func filterEmptyResponses(responses []*MethodResponse) []*MethodResponse {
	var result []*MethodResponse

	for _, resp := range responses {
		if resp == nil {
			continue
		}

		result = append(result, resp)
	}

	return result
}

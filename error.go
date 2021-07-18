package jsonrpc

const (
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603

	ServerErrorStartCodeValue = -32099

	parseErrorMessage     = "Parse error"
	invalidRequestMessage = "Invalid Request"
	methodNotFoundMessage = "Method not found"
	invalidParamsMessage  = "Invalid params"
	internalErrorMessage  = "Internal error"
)

var builtInErrs map[int]string

func init() {
	builtInErrs = map[int]string{
		ParseErrorCode:     parseErrorMessage,
		InvalidRequestCode: invalidRequestMessage,
		MethodNotFoundCode: methodNotFoundMessage,
		InvalidParamsCode:  invalidParamsMessage,
		InternalErrorCode:  internalErrorMessage,
	}
}

type ServerError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func MakeError(code int, message string, data error) *ServerError {
	err := &ServerError{
		Code:    code,
		Message: message,
		Data:    data,
	}

	if data != nil {
		err.Data = data.Error()
	}

	return err
}

func MakeBuiltInError(code int, data error) *ServerError {
	msg, ok := builtInErrs[code]
	if !ok {
		return MakeError(InternalErrorCode, internalErrorMessage, data)
	}

	return MakeError(code, msg, data)
}

func throwError(err *ServerError) {
	panic(err)
}

func makeErrorAndThrow(code int, data error) {
	throwError(MakeBuiltInError(code, data))
}

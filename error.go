package jsonrpc

const (
	module_code_multiplier = 1000

	jsonrpc_module_code = 0
)

type Error struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ServerError interface {
	GetCode() int
	GetMessage() string
	GetData() *string
	ToError() *Error
}

type serverErrorImpl struct {
	code    int
	message string
	data    error
}

func (e *serverErrorImpl) GetCode() int {
	return e.code
}

func (e *serverErrorImpl) GetMessage() string {
	return e.message
}

func (e *serverErrorImpl) GetData() *string {
	if e.data == nil {
		return nil
	}
	v := e.data.Error()
	return &v
}

func (e *serverErrorImpl) ToError() *Error {
	return &Error{
		Code:    e.GetCode(),
		Message: e.GetMessage(),
		Data:    e.GetData(),
	}
}

func MakeError(code int, message string, data error) ServerError {
	return &serverErrorImpl{
		code:    code,
		message: message,
		data:    data,
	}
}

func makeModuleErrorCode(module int, errorCode int) int {
	return module*module_code_multiplier + errorCode
}

func MakeModuleError(module int, errorCode int, msg string, data error) ServerError {
	code := makeModuleErrorCode(module, errorCode)
	return MakeError(code, msg, data)
}

func throwError(err ServerError) {
	panic(err)
}

func makeErrorAndThrow(code int, message string, data error) {
	handlerErrorCode := makeModuleErrorCode(jsonrpc_module_code, code)
	throwError(MakeError(handlerErrorCode, message, data))
}

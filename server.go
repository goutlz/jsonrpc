package jsonrpc

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/goutlz/servr"
)

func NewServer(opts *ServerOpts) servr.Server {
	router := mux.NewRouter()
	router.Handle("/jsonrpc", createJsonRpcHandler(opts.MethodHandlers)).Methods("POST")

	var handler http.Handler = router
	if opts.Cors.isEnabled() {
		handler = handlers.CORS(opts.Cors.listHandlersCorsOptions()...)(router)
	}

	return servr.New(opts.Addr, handler)
}

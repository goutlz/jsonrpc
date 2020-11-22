package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/goutlz/errz"
	"github.com/goutlz/servr"

	"github.com/gorilla/mux"
)

type Server interface {
	Stop() error
}

type serverWrap struct {
	lock    sync.RWMutex
	server  *http.Server
	stopped bool
}

func (s *serverWrap) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopped {
		return errz.New("Server already stopped")
	}

	s.stopped = true

	err := s.server.Shutdown(context.Background())
	if err != nil {
		return errz.Wrap(err, "Failed to stop server")
	}

	return nil
}

func (s *serverWrap) isStopped() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.stopped
}

func NewServer(addr string, handlers map[string]RouteHandler) Server {
	router := mux.NewRouter()

	versionRouter := router.PathPrefix("/api").Subrouter()
	for route, handler := range handlers {
		versionRouter.HandleFunc(fmt.Sprintf("/%s", route), createJsonRpcHandler(handler)).Methods("POST")
	}

	return servr.New(addr, router)
}

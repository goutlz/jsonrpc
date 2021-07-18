package jsonrpc

import (
	"github.com/gorilla/handlers"
	"github.com/goutlz/errz"
	"net/http"
)

type ServerOpts struct {
	Addr           string
	MethodHandlers MethodHandlers
	Cors           *CorsOpts
}

type CorsOpts struct {
	Enabled bool
	Origins []string
	Headers []string
	Methods []string
}

func (c *CorsOpts) isEnabled() bool {
	return c != nil && c.Enabled
}

func (c *CorsOpts) listHandlersCorsOptions() []handlers.CORSOption {
	return []handlers.CORSOption{
		handlers.AllowedOrigins(c.getAllowedOrigins()),
		handlers.AllowedHeaders(c.getAllowedHeaders()),
		handlers.AllowedMethods(c.getAllowedMethods()),
	}
}

func (c *CorsOpts) getAllowedOrigins() []string {
	if len(c.Origins) == 0 {
		return []string{"*"}
	}

	return c.Origins
}

func (c *CorsOpts) getAllowedHeaders() []string {
	if len(c.Origins) == 0 {
		return []string{"X-Requested-With", "Content-Type"}
	}

	return c.Origins
}

func (c *CorsOpts) getAllowedMethods() []string {
	if len(c.Origins) == 0 {
		return []string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}
	}

	return c.Origins
}

type ClientOpts struct {
	HttpClient *http.Client
	IdFactory  IdFactory
	Url        string
}

func (c *ClientOpts) validate() error {
	if c.HttpClient == nil {
		return errz.New("Http client required")
	}

	return nil
}

func (c *ClientOpts) getIdFactory() IdFactory {
	if c.IdFactory == nil {
		return defaultIdFactory
	}

	return c.IdFactory
}

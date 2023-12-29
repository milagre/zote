package zapi

import (
	"context"
	"io"
	"net/http"
)

type HandleFunc func(Request) ResponseBuilder

type Method struct {
	Handler HandleFunc
}

type Methods map[string]Method

type ResponseBuilder interface {
	Status() int
	Headers() http.Header
	Body() io.Reader
}

type Request interface {
	HTTPRequest() *http.Request

	Context() context.Context
	AddContextValue(key any, val any)

	Body() ([]byte, error)
	Header() http.Header
	Method() string
	Query(key string) string
	Param(p string) string
	//Params() map[string][]string
}

type Route interface {
	Name() string
	Path() string
	Methods() Methods
}

type AuthorizingRoute interface {
	Authorize(req Request) ResponseBuilder
}

package api

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
	Context() context.Context
	Header() http.Header
	Method() string
}

type Route interface {
	Path() string
	Methods() Methods
}

type AuthorizingRoute interface {
	Authorize(req Request) ResponseBuilder
}

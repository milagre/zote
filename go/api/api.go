package api

import (
	"context"
	"io"
	"net/http"
)

type MethodFunc func(Request) ResponseBuilder
type Methods map[string]MethodFunc

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

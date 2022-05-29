package api

import (
	"context"
	"net/http"
	"strings"
)

var _ Request = &request{}

type request struct {
	request *http.Request
}

func (r *request) Context() context.Context {
	return r.request.Context()
}

func (r *request) Header() http.Header {
	return r.request.Header.Clone()
}

func (r *request) Method() string {
	return strings.ToUpper(r.request.Method)
}

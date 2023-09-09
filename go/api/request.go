package api

import (
	"context"
	"net/http"
	"strings"
)

var _ Request = &request{}

type request struct {
	request *http.Request
	route   Route
	params  map[string][]string
}

func (r *request) HTTPRequest() *http.Request {
	return r.request
}

func (r *request) Context() context.Context {
	return r.request.Context()
}

func (r *request) AddContextValue(key any, val any) {
	ctx := r.request.Context()
	ctx = context.WithValue(ctx, key, val)
	r.request = r.request.WithContext(ctx)
}

func (r *request) Header() http.Header {
	return r.request.Header.Clone()
}

func (r *request) Method() string {
	return strings.ToUpper(r.request.Method)
}

func (r *request) Param(p string) string {
	vals, ok := r.params[p]
	if !ok {
		return ""
	}

	if len(vals) == 1 {
		return vals[0]
	}

	return ""
}

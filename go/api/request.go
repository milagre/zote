package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var _ Request = &request{}

type request struct {
	request   *http.Request
	route     Route
	params    map[string][]string
	bodyCache []byte
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

func (r *request) Body() ([]byte, error) {
	if r.bodyCache != nil {
		return r.bodyCache, nil
	}

	r.bodyCache = make([]byte, 0)

	data, err := io.ReadAll(r.HTTPRequest().Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %w", err)
	}

	r.bodyCache = data
	return r.bodyCache, nil
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

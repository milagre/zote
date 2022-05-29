package api

import (
	"io"
	"net/http"
)

type simpleResponse struct {
	status  int
	headers http.Header
	body    io.Reader
}

func (r simpleResponse) Status() int {
	return r.status
}

func (r simpleResponse) Headers() http.Header {
	if r.headers == nil {
		return http.Header{}
	}
	return r.headers
}

func (r simpleResponse) Body() io.Reader {
	return r.body
}

func Response200OK() ResponseBuilder {
	return simpleResponse{
		status: http.StatusOK,
	}
}

func Response401Unauthorized() ResponseBuilder {
	return simpleResponse{
		status: http.StatusUnauthorized,
	}
}

func Response404NotFound() ResponseBuilder {
	return simpleResponse{
		status: http.StatusNotFound,
	}
}

func Response405MethodNotAllowed() ResponseBuilder {
	return simpleResponse{
		status: http.StatusMethodNotAllowed,
	}
}

func Response500InternalServerError() ResponseBuilder {
	return simpleResponse{
		status: http.StatusInternalServerError,
	}
}

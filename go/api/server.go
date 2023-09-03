package api

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	zotelog "github.com/milagre/zote/go/log"
)

type Server struct {
	mux      *http.ServeMux
	logger   zotelog.Logger
	defaults struct {
		methodNotAllowed    func() ResponseBuilder
		notFound            func() ResponseBuilder
		internalServerError func() ResponseBuilder
	}
	routes  map[string]Route
	parents map[string]Route

	// Set when listen is called
	server *http.Server

	// Closed when Shutdown returns
	shutdown chan struct{}
}

func NewServer(logger zotelog.Logger, routes []Route) (*Server, error) {
	server := &Server{
		logger:   logger,
		mux:      http.NewServeMux(),
		shutdown: make(chan struct{}),
		routes:   map[string]Route{},
		parents:  map[string]Route{},
	}

	for _, route := range routes {
		err := server.mount(route)
		if err != nil {
			return nil, fmt.Errorf("error mounting route(s): %w", err)
		}
	}

	if server.defaults.methodNotAllowed == nil {
		server.defaults.methodNotAllowed = func() ResponseBuilder {
			return Response405MethodNotAllowed()
		}
	}

	if server.defaults.notFound == nil {
		server.defaults.notFound = func() ResponseBuilder {
			return Response404NotFound()
		}
	}

	if server.defaults.internalServerError == nil {
		server.defaults.internalServerError = func() ResponseBuilder {
			return Response500InternalServerError()
		}
	}

	return server, nil
}

func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{Addr: addr, Handler: s.mux}
	err := s.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}
	<-s.shutdown
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	defer close(s.shutdown)
	return s.server.Shutdown(ctx)
}

func (s *Server) mount(route Route) error {
	path := route.Path()
	path = strings.TrimRight(path, "/")
	if path == "" || path[0] != '/' {
		path = "/" + path
	}

	parent := "/"

	last := strings.LastIndex(path, "/")
	if last != 0 {
		/*
			if path != "/" {
				return fmt.Errorf("root route path must be '' or '/', following is invalid: %s", route.Path())
			}
		} else {*/
		parent = path[0:last]
	}

	s.routes[path] = route

	if path != "/" {
		parentRoute, ok := s.routes[parent]
		if !ok {
			return fmt.Errorf("parent route for '%s' not found (expecting '%s'); mount parents first", route.Path(), parent)
		}

		s.parents[path] = parentRoute
	}

	s.logger.Infof("Mounting %s (parent %s)", path, parent)
	s.mux.HandleFunc(path, handle(s, route))
	return nil
}

func handle(server *Server, route Route) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		defer func() {
			_, _ = ioutil.ReadAll(r.Body)
			_ = r.Body.Close()
		}()

		start := time.Now()
		methods := route.Methods()
		requestID := uuid.New().String()
		logger := server.logger.WithField("request", requestID)
		r = r.WithContext(zotelog.Context(r.Context(), logger))

		access := logger.WithFields(zotelog.Fields{
			"component": "access",
			"method":    r.Method,
			"url":       r.URL.String(),
		})
		access.Info("Starting")

		req := &request{
			request: r,
		}
		var resp ResponseBuilder

		defer func() {
			if r := recover(); r != nil {
				logger.Warnf("Panic: %v", r)
			}

			if resp == nil {
				resp = server.defaults.internalServerError()
			}

			len := write(server, logger, rw, resp, false)

			access.WithFields(zotelog.Fields{
				"status":   resp.Status(),
				"duration": time.Since(start),
				"length":   len,
			}).Info("Complete")
		}()

		parents := []Route{route}
		parent := route
		ok := true
		fmt.Println(parent.Path())
		for ok {
			parent, ok = server.parents[parent.Path()]
			if ok {
				parents = append(parents, parent)
				fmt.Println(parent.Path())
			}
		}

		for i := len(parents) - 1; i >= 0; i-- {
			parent := parents[i]
			if auth, ok := parent.(AuthorizingRoute); ok {
				fmt.Println("test")
				resp = auth.Authorize(req)
			}
		}

		method, ok := methods[r.Method]
		if !ok {
			resp = server.defaults.methodNotAllowed()
		} else if r.URL.Path != route.Path() {
			resp = server.defaults.notFound()
		} else {
			resp = method.Handler(req)
		}
	}
}

func write(server *Server, logger zotelog.Logger, rw http.ResponseWriter, resp ResponseBuilder, truncateOnFail bool) int {
	raw := resp.Body()

	var body []byte
	if raw != nil {
		var err error
		body, err = io.ReadAll(raw)
		if err != nil {
			if !truncateOnFail {
				logger.Warnf("Overriding response with Internal Server Error due to error reading response body: %v", err)
				resp = server.defaults.internalServerError()
				return write(server, logger, rw, resp, true)
			}

			logger.Warnf("Truncating response due to error while reading response body: %v", err)
			body = []byte{}
		}
	}

	headers := resp.Headers().Clone()
	headers.Add("Content-Length", fmt.Sprintf("%d", len(body)))

	for key, values := range headers {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	rw.WriteHeader(resp.Status())
	_, err := rw.Write(body)
	if err != nil {
		logger.Warnf("Truncating response body due to error while reading overriden Internal Server Error response body, truncating body: %v", err)
	}

	return len(body)
}

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/milagre/zote/go/log"
)

type Server interface {
	ListenAndServe(addr string) error
	Shutdown(ctx context.Context) error
}

type server struct {
	handler  *handlerTree
	logger   log.Logger
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

func NewServer(logger log.Logger, routes []Route) (*server, error) {
	server := &server{
		logger:   logger,
		handler:  &handlerTree{},
		shutdown: make(chan struct{}),
		routes:   map[string]Route{},
		parents:  map[string]Route{},
	}
	server.handler.server = server

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

type handlerTree struct {
	// routes[path]handler
	server *server
	root   *handler
}

type handler struct {
	part     string
	param    *string
	route    Route
	children map[string]*handler
}

func (h *handlerTree) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	defer func() {
		_, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
	}()

	start := time.Now()
	requestID := uuid.New().String()
	logger := h.server.logger.WithField("request", requestID)
	r = r.WithContext(log.Context(r.Context(), logger))

	access := logger.WithFields(log.Fields{
		"component": "access",
		"url":       r.URL.String(),
		"method":    r.Method,
	})

	// Return when done, this will wrote the assigned response (or default) to caller
	var resp ResponseBuilder
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("Panic: %+v; %s", r, debug.Stack())
		}

		if resp == nil {
			resp = h.server.defaults.internalServerError()
		}

		len := write(h.server, logger, rw, resp, false)

		access.WithFields(log.Fields{
			"status":   resp.Status(),
			"duration": time.Since(start),
			"length":   len,
		}).Info("Complete")
	}()

	// If a route is found, this will call it
	execute := func(parents []Route, route Route, params map[string][]string) {
		access = access.WithFields(log.Fields{
			"route": route.Name(),
		})
		access.Info("Starting")

		req := &request{
			request: r,
			route:   route,
			params:  params,
		}

		for _, parent := range parents {
			if auth, ok := parent.(AuthorizingRoute); ok {
				resp = auth.Authorize(req)
				if resp != nil {
					return
				}
			}
		}

		method, ok := route.Methods()[r.Method]
		if !ok {
			resp = h.server.defaults.methodNotAllowed()
		} else {
			resp = method.Handler(req)
		}
	}

	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	params := map[string][]string{}

	// Root resource requested, short circuit
	if len(parts) == 1 && parts[0] == "" {
		execute([]Route{}, h.root.route, params)
		return
	}

	parents := []Route{h.root.route}
	current := h.root
	for _, part := range parts {
		child, ok := current.children[part]
		if !ok {
			// Dynamic URL part
			child, ok = current.children[""]
			if !ok {
				access.Info("Starting")
				resp = h.server.defaults.notFound()
				return
			}
		}

		// Dynamic URL part
		if child.param != nil {
			param := *child.param
			params[param] = append(params[param], part)
		}

		parents = append(parents, child.route)
		current = child
	}

	execute(parents, current.route, params)
}

func isParam(p string) (string, bool) {
	length := len(p)
	if length < 3 {
		return "", false
	}

	if p[0] != '{' || p[length-1] != '}' {
		return "", false
	}

	name := p[1 : length-1]

	for _, c := range name {
		if c == '{' || c == '}' {
			return "", false
		}
	}

	return name, true
}

func (h *handlerTree) add(route Route) error {
	path := route.Path()
	path = strings.Trim(path, "/")

	if path == "" {
		h.root = &handler{
			part:     "",
			route:    route,
			children: map[string]*handler{},
		}
	} else if h.root != nil {
		parts := strings.Split(path, "/")
		last := len(parts) - 1

		current := h.root

		for _, part := range parts[0:last] {
			if _, ok := isParam(part); ok {
				part = ""
			}

			child, ok := current.children[part]
			if !ok {
				return fmt.Errorf("parent route for '%s' not found; mount parents before children", route.Path())
			}

			current = child
		}

		lastPart := parts[last]
		pname, param := isParam(lastPart)
		if param {
			lastPart = ""
		}

		if _, ok := current.children[lastPart]; ok {
			return fmt.Errorf("dynamic child already set on parent, route '%s' cannot be added", route.Path())
		}

		routeHandler := &handler{
			part:     lastPart,
			route:    route,
			children: map[string]*handler{},
		}

		if param {
			routeHandler.param = &pname
		}

		current.children[lastPart] = routeHandler

	} else {
		return fmt.Errorf("root route not yet mounted, mount the root route first, then children after")
	}

	h.server.logger.Infof("Mounted %s", path)
	return nil
}

func (s *server) ListenAndServe(addr string) error {
	s.server = &http.Server{Addr: addr, Handler: s.handler}
	err := s.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}
	<-s.shutdown
	return nil
}

func (s *server) Shutdown(ctx context.Context) error {
	defer close(s.shutdown)
	return s.server.Shutdown(ctx)
}

func (s *server) mount(route Route) error {
	err := s.handler.add(route)
	if err != nil {
		return fmt.Errorf("mounting route: %w", err)
	}

	return nil
}

func write(s *server, logger log.Logger, rw http.ResponseWriter, resp ResponseBuilder, truncateOnFail bool) int {
	raw := resp.Body()

	var body []byte
	if raw != nil {
		var err error
		body, err = io.ReadAll(raw)
		if err != nil {
			if !truncateOnFail {
				logger.Warnf("Overriding response with Internal Server Error due to error reading response body: %v", err)
				resp = s.defaults.internalServerError()
				return write(s, logger, rw, resp, true)
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

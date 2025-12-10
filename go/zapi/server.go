package zapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zstats"
)

type Server interface {
	ListenAndServe(addr string) error
	Shutdown(ctx context.Context) error
}

type server struct {
	rootContext context.Context
	handler     *handlerTree
	middleware  []Middleware
	defaults    struct {
		optionsRequest      HandleFunc
		methodNotAllowed    HandleFunc
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

type Option func(s *server)

func ServerOptionDefaultOptionsRequest(f HandleFunc) Option {
	return func(s *server) {
		s.defaults.optionsRequest = f
	}
}

func NewServer(ctx context.Context, routes []Route, options ...Option) (*server, error) {
	server := &server{
		rootContext: ctx,
		handler:     &handlerTree{},
		shutdown:    make(chan struct{}),
		routes:      map[string]Route{},
		parents:     map[string]Route{},
		middleware:  []Middleware{},
	}
	server.handler.server = server

	for _, route := range routes {
		err := server.mount(route)
		if err != nil {
			return nil, fmt.Errorf("error mounting route(s): %w", err)
		}
	}

	if server.defaults.methodNotAllowed == nil {
		server.defaults.methodNotAllowed = func(req Request) ResponseBuilder {
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

	if server.defaults.optionsRequest == nil {
		server.defaults.optionsRequest = func(req Request) ResponseBuilder {
			return server.defaults.methodNotAllowed(req)
		}
	}

	for _, opt := range options {
		opt(server)
	}

	return server, nil
}

func (s *server) AddMiddleware(m Middleware) {
	s.middleware = append(s.middleware, m)
}

func (s *server) middlewareChain(handle HandleFunc) HandleFunc {
	var reversed []Middleware
	for _, m := range s.middleware {
		reversed = append([]Middleware{m}, reversed...)
	}

	target := handle
	for _, m := range reversed {
		target = func(m Middleware, next HandleFunc) HandleFunc {
			return func(req Request) ResponseBuilder {
				return m(req, next)
			}
		}(m, target)
	}

	return func(req Request) ResponseBuilder {
		return target(req)
	}
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

	requestContext := r.Context()

	logger := zlog.FromContext(h.server.rootContext)
	logger = logger.WithField("request", requestID)
	requestContext = zlog.Context(requestContext, logger)

	stats := zstats.FromContext(h.server.rootContext)
	stats = stats.WithPrefix("zapi")
	requestContext = zstats.Context(requestContext, stats)

	r = r.WithContext(requestContext)

	access := logger.WithFields(zlog.Fields{
		"component": "access",
		"url":       r.URL.String(),
		"method":    r.Method,
	})

	// Return when done, this will write the assigned response (or default) to caller
	var resp ResponseBuilder
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("Panic: %+v; %s", r, debug.Stack())
		}

		if resp == nil {
			resp = h.server.defaults.internalServerError()
		}

		len := write(h.server, logger, rw, resp, false)

		respLog := access.WithFields(zlog.Fields{
			"status":   resp.Status(),
			"duration": time.Since(start),
			"length":   len,
		})
		stats.WithTags(zstats.Tags{
			"status": fmt.Sprintf("%d", resp.Status()),
		}).Count("responses", 1)

		logFunc := respLog.Info
		if resp.Status()/100 == 5 {
			logFunc = respLog.Error
		}
		logFunc("Complete")
	}()

	// If a route is found, this will call it
	execute := func(allRoutes []Route, targetRoute Route, params map[string][]string) {
		access = access.WithFields(zlog.Fields{
			"route": targetRoute.Name(),
		})
		access.Info("Starting")
		stats = stats.WithTags(zstats.Tags{
			"method": r.Method,
			"route":  targetRoute.Name(),
			"path":   targetRoute.Path(),
		})

		req := &request{
			request: r,
			route:   targetRoute,
			params:  params,
		}

		method, ok := targetRoute.Methods()[r.Method]
		var handler HandleFunc
		if !ok {
			if r.Method == http.MethodOptions {
				handler = h.server.defaults.optionsRequest
			} else {
				handler = h.server.defaults.methodNotAllowed
			}
		} else {
			stats.Count("requests", 1)

			handler = func(Request) (res ResponseBuilder) {
				stats.Timer("handler", func() {
					res = method.Handler(req)
				})
				return res
			}

			// We only authorize calls that hit a handler,
			// calls that hit a default can execute alone
			for _, route := range allRoutes {
				if auth, ok := route.(AuthorizingRoute); ok {
					authResp := auth.Authorize(req)
					if authResp != nil {
						handler = func(req Request) ResponseBuilder {
							return authResp
						}
					}
				}
			}
		}

		resp = h.server.middlewareChain(handler)(req)
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

	zlog.FromContext(h.server.rootContext).Infof("Mounted %s", path)
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

func write(s *server, logger zlog.Logger, rw http.ResponseWriter, resp ResponseBuilder, truncateOnFail bool) int {
	raw := resp.Body()

	var body []byte
	if raw != nil {
		var err error
		body, err = io.ReadAll(raw)
		if err != nil {
			if !truncateOnFail {
				logger.Errorf("Overriding response with Internal Server Error due to error reading response body: %v", err)
				resp = s.defaults.internalServerError()
				return write(s, logger, rw, resp, true)
			}

			logger.Errorf("Truncating response due to error while reading response body: %v", err)
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

	status := resp.Status()
	rw.WriteHeader(status)

	if status == http.StatusNoContent {
		return 0
	}

	_, err := rw.Write(body)
	if err != nil {
		logger.Warnf("Error writing response body for request: %v", err)
	}

	return len(body)
}

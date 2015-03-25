package testmux

import (
	"fmt"
	"net/http"
	"testing"
)

// Router registers routes to be matched and dispatches the associated
// handler. It will track handled requests to ensure that all registered routes
// are invoked and that they are received in the same order in which they were
// originally registered.
//
// It implements the http.Hander interface.
type Router struct {
	routes []route
	index  int
	errors []routeError
}

type route struct {
	method  string
	path    string
	handler func(http.ResponseWriter, *http.Request)
	visited bool
}

type errorType int

const (
	unvisited errorType = iota
	unexpected
	disorderly
)

type routeError struct {
	fault  errorType
	method string
	path   string
}

// RegisterFunc registers a handler function for the given request method and
// path.
func (r *Router) RegisterFunc(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	rte := route{method: method, path: path, handler: handler}
	r.routes = append(r.routes, rte)
}

// RegisterResp registers a static status code and body string to be returned
// for the given request method and path.
func (r *Router) RegisterResp(method, path string, status int, body string) {
	rte := route{
		method: method,
		path:   path,
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			fmt.Fprintln(w, body)
		},
	}

	r.routes = append(r.routes, rte)
}

// ServeHTTP dispatches the handler registered in the matched route and tracks
// whether the route was requested in the correct order.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rte, ordered := r.match(req.Method, req.URL.Path)

	if rte != nil {
		rte.execute(w, req)

		if !ordered {
			r.addError(disorderly, req.Method, req.URL.Path)
		}
	} else {
		http.NotFound(w, req)
		r.addError(unexpected, req.Method, req.URL.Path)
	}

	r.index++
}

// AssertVisited asserts that all of the registered routes were visited.
// Conditions that will cause the assertion to fail include: requests received
// for unregistered routes, and routes registered but never requested.
//
// Returns whether the assertion was successful (true) or not (false).
func (r *Router) AssertVisited(t *testing.T) bool {
	return r.assert(t, false)
}

// AssertVisited asserts that all of the registered routes were visited in the
// correct order. Conditions that will cause the assertion to fail include:
// requests received out of order, requests received for unregistered routes,
// and routes registered but never requested.
//
// Returns whether the assertion was successful (true) or not (false).
func (r *Router) AssertVisitedInOrder(t *testing.T) bool {
	return r.assert(t, true)
}

func (r *Router) assert(t *testing.T, ordered bool) bool {
	success := true

	for _, rte := range r.routes {
		if !rte.visited {
			r.addError(unvisited, rte.method, rte.path)
		}
	}

	for _, err := range r.errors {
		if err.fault == disorderly && !ordered {
			continue
		}

		assertError(t, err)
		success = false
	}

	return success
}

// Adds an error to the internal collection.
func (r *Router) addError(fault errorType, method, path string) {
	r.errors = append(r.errors, routeError{fault, method, path})
}

// Given an HTTP method and request path, looks for a matching handler which
// has not already been visited. Returns the handler along with a flag
// indicating whether or not the handler is being invoked in the correct
// order.
func (r *Router) match(method, path string) (*route, bool) {
	for i, rte := range r.routes {
		if !rte.visited && rte.method == method && rte.path == path {
			return &r.routes[i], (i == r.index)
		}
	}

	return nil, false
}

// Execute the handler associated with the route and mark it as visited.
func (rte *route) execute(w http.ResponseWriter, req *http.Request) {
	rte.handler(w, req)
	rte.visited = true
}

func assertError(t *testing.T, err routeError) {
	switch err.fault {
	case unvisited:
		t.Errorf("Unvisited route: %s %s", err.method, err.path)
	case unexpected:
		t.Errorf("Unexpected request: %s %s", err.method, err.path)
	case disorderly:
		t.Errorf("Request out of order: %s %s", err.method, err.path)
	}
}

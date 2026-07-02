// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "strings"

// Router holds an ordered set of routes and resolves a request to a match. Route
// order is significant: the first route whose method and path both match wins,
// mirroring Grape's first-declared-first-served dispatch.
type Router struct {
	routes []*Route
}

// NewRouter returns an empty Router.
func NewRouter() *Router { return &Router{} }

// Add appends a route. Routes are matched in insertion order.
func (rt *Router) Add(r *Route) { rt.routes = append(rt.routes, r) }

// Get/Post/Put/Patch/Delete/Head add a route for the corresponding verb and
// return it, so a handler can be attached fluently.
func (rt *Router) Get(pattern string, h any) *Route    { return rt.add("GET", pattern, h) }
func (rt *Router) Post(pattern string, h any) *Route   { return rt.add("POST", pattern, h) }
func (rt *Router) Put(pattern string, h any) *Route    { return rt.add("PUT", pattern, h) }
func (rt *Router) Patch(pattern string, h any) *Route  { return rt.add("PATCH", pattern, h) }
func (rt *Router) Delete(pattern string, h any) *Route { return rt.add("DELETE", pattern, h) }
func (rt *Router) Head(pattern string, h any) *Route   { return rt.add("HEAD", pattern, h) }

func (rt *Router) add(method, pattern string, h any) *Route {
	r := NewRoute(method, pattern, h)
	rt.Add(r)
	return r
}

// Routes returns the routes in declaration order.
func (rt *Router) Routes() []*Route { return rt.routes }

// MatchStatus is the routing decision for a request.
type MatchStatus int

const (
	// StatusOK: a route matched both method and path.
	StatusOK MatchStatus = iota
	// StatusNotFound (404): no route's path matched.
	StatusNotFound
	// StatusMethodNotAllowed (405): the path matched at least one route but not
	// for the request method.
	StatusMethodNotAllowed
)

// Match resolves a request. On StatusOK, Route and Params are the matched route
// and its captured path params. On StatusMethodNotAllowed, Allowed lists the
// methods the path does accept (for the Allow header). GET implicitly satisfies
// a HEAD request (Grape/Rack semantics).
type Match struct {
	Status  MatchStatus
	Route   *Route
	Params  map[string]string
	Allowed []string
}

// Match resolves (method, path) to a routing decision. HTTP status content
// negotiation (406) is a separate step; see [Router.Negotiate].
func (rt *Router) Match(method, path string) Match {
	method = strings.ToUpper(method)
	var allowed []string
	seen := map[string]bool{}
	for _, r := range rt.routes {
		params, ok := r.matchPath(path)
		if !ok {
			continue
		}
		if r.methodMatches(method) {
			return Match{Status: StatusOK, Route: r, Params: params}
		}
		// Path matched, method did not. Only a concrete-method route can reach
		// here (an empty / "*" method always matches), so r.Method is a real verb;
		// record it once for the Allow header.
		if m := r.Method; !seen[m] {
			seen[m] = true
			allowed = append(allowed, m)
		}
	}
	if len(allowed) > 0 {
		return Match{Status: StatusMethodNotAllowed, Allowed: allowed}
	}
	return Match{Status: StatusNotFound}
}

// methodMatches reports whether the route's method accepts the request method.
// An empty or "*" route method matches any verb; a GET route also answers HEAD.
func (r *Route) methodMatches(method string) bool {
	rm := strings.ToUpper(r.Method)
	if rm == "" || rm == "*" {
		return true
	}
	if rm == method {
		return true
	}
	if method == "HEAD" && rm == "GET" {
		return true
	}
	return false
}

// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"reflect"
	"testing"
)

func TestRouteMatchPath(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    map[string]string
		ok      bool
	}{
		{"/users/:id", "/users/42", map[string]string{"id": "42"}, true},
		{"/users/:id", "/users", nil, false},
		{"/users/:id", "/users/42/extra", nil, false},
		{"/users", "/users/", map[string]string{}, true},
		{"/", "/", map[string]string{}, true},
		{"/", "/x", nil, false},
		{"/a/b", "/a/b", map[string]string{}, true},
		{"/a/b", "/a/c", nil, false},
		{"/files/*path", "/files/a/b/c.txt", map[string]string{"path": "a/b/c.txt"}, true},
		{"/files/*path", "/files", map[string]string{"path": ""}, true},
		{"/files/*path", "/other", nil, false},
		{"/users/:id.:format", "/users/42.json", map[string]string{"id": "42", "format": "json"}, true},
		{"/users/:id.:format", "/users/42", nil, false},
		{"/report.:format", "/report.xml", map[string]string{"format": "xml"}, true},
		{"/report.:format", "/summary.xml", nil, false},
		{"/report.:format", "/report", nil, false},
		{"users/:id", "users/7", map[string]string{"id": "7"}, true},
	}
	for _, c := range cases {
		r := NewRoute("GET", c.pattern, nil)
		got, ok := r.matchPath(c.path)
		if ok != c.ok {
			t.Errorf("%s vs %s: ok=%v want %v", c.pattern, c.path, ok, c.ok)
			continue
		}
		if ok && !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s vs %s: params=%v want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestSplitFormat(t *testing.T) {
	cases := []struct {
		in           string
		base, format string
		ok           bool
	}{
		{"a.json", "a", "json", true},
		{"a.b.json", "a.b", "json", true},
		{"noext", "", "", false},
		{".json", "", "", false},
		{"a.", "", "", false},
	}
	for _, c := range cases {
		b, f, ok := splitFormat(c.in)
		if b != c.base || f != c.format || ok != c.ok {
			t.Errorf("splitFormat(%q) = (%q,%q,%v) want (%q,%q,%v)", c.in, b, f, ok, c.base, c.format, c.ok)
		}
	}
}

func TestRouterMatch(t *testing.T) {
	rt := NewRouter()
	rt.Get("/users/:id", "getUser")
	rt.Post("/users", "createUser")
	rt.Put("/users/:id", "putUser")
	rt.Patch("/users/:id", "patchUser")
	rt.Delete("/users/:id", "delUser")
	rt.Head("/health", "head")

	m := rt.Match("GET", "/users/42")
	if m.Status != StatusOK || m.Route.Handler != "getUser" || m.Params["id"] != "42" {
		t.Fatalf("GET /users/42 -> %+v", m)
	}
	if m := rt.Match("POST", "/users"); m.Status != StatusOK || m.Route.Handler != "createUser" {
		t.Fatalf("POST /users -> %+v", m)
	}
	// HEAD satisfied by a GET route.
	rt2 := NewRouter()
	rt2.Get("/g", "g")
	if m := rt2.Match("HEAD", "/g"); m.Status != StatusOK {
		t.Fatalf("HEAD /g -> %+v", m)
	}
	// 405: path matches, method does not.
	if m := rt.Match("OPTIONS", "/users/42"); m.Status != StatusMethodNotAllowed {
		t.Fatalf("OPTIONS /users/42 -> %+v", m)
	} else if len(m.Allowed) == 0 {
		t.Fatalf("expected Allowed methods, got none")
	}
	// 404: no path matches.
	if m := rt.Match("GET", "/nope"); m.Status != StatusNotFound {
		t.Fatalf("GET /nope -> %+v", m)
	}
}

func TestRouterMethodWildcard(t *testing.T) {
	rt := NewRouter()
	rt.Add(NewRoute("", "/any", "anyMethod"))
	rt.Add(NewRoute("*", "/star", "starMethod"))
	if m := rt.Match("DELETE", "/any"); m.Status != StatusOK || m.Route.Handler != "anyMethod" {
		t.Fatalf("DELETE /any -> %+v", m)
	}
	if m := rt.Match("PATCH", "/star"); m.Status != StatusOK {
		t.Fatalf("PATCH /star -> %+v", m)
	}
	// 405 dedup + "*" recorded in Allowed.
	rt2 := NewRouter()
	rt2.Add(NewRoute("", "/x/:id", "h1"))
	m := rt2.Match("GET", "/x/1")
	if m.Status != StatusOK {
		t.Fatalf("empty-method route should match any: %+v", m)
	}
}

func TestRouterAllowedDedup(t *testing.T) {
	rt := NewRouter()
	rt.Post("/r", "a")
	rt.Post("/r", "b") // same method, same path: Allowed must not duplicate
	m := rt.Match("GET", "/r")
	if m.Status != StatusMethodNotAllowed {
		t.Fatalf("want 405, got %+v", m)
	}
	if len(m.Allowed) != 1 || m.Allowed[0] != "POST" {
		t.Fatalf("Allowed = %v, want [POST]", m.Allowed)
	}
}

func TestRoutesAccessor(t *testing.T) {
	rt := NewRouter()
	r := rt.Get("/x", nil)
	if got := rt.Routes(); len(got) != 1 || got[0] != r {
		t.Fatalf("Routes() = %v", got)
	}
}

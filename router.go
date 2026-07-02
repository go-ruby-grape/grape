// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "strings"

// Route is a single Grape endpoint: an HTTP method plus a compiled path pattern.
// A pattern is a slash-separated list of segments; a segment may be a literal, a
// named capture (":name"), or a trailing wildcard ("*name") that greedily
// swallows the remainder of the path. A ".:format" suffix on the final segment
// binds the request extension (e.g. "/users/:id.:format").
type Route struct {
	Method  string // upper-case HTTP verb ("GET", "POST", …); "" or "*" matches any
	Pattern string // the raw declared pattern, e.g. "/users/:id"
	segs    []segment
	// Handler is an opaque host reference to the endpoint body. The router never
	// runs it; it is handed back on a match so the host (rbgo) can invoke it.
	Handler any
}

type segKind int

const (
	segLiteral segKind = iota
	segParam
	segWildcard
)

type segment struct {
	kind   segKind
	name   string // capture name for segParam / segWildcard
	lit    string // literal text for segLiteral
	fmtCap bool   // this segment carries a trailing ".:format" capture
}

// NewRoute compiles a method + pattern into a Route. The pattern is normalised:
// a leading slash is optional, and a trailing slash is ignored (Grape treats
// "/users" and "/users/" alike).
func NewRoute(method, pattern string, handler any) *Route {
	r := &Route{
		Method:  strings.ToUpper(method),
		Pattern: pattern,
		Handler: handler,
	}
	r.segs = compilePattern(pattern)
	return r
}

func compilePattern(pattern string) []segment {
	p := strings.Trim(pattern, "/")
	if p == "" {
		return nil
	}
	parts := strings.Split(p, "/")
	segs := make([]segment, 0, len(parts))
	for _, part := range parts {
		segs = append(segs, compileSegment(part))
	}
	return segs
}

// compileSegment turns one path component into a segment. A ".:format" tail on a
// literal or param segment sets fmtCap so the matcher captures the extension.
func compileSegment(part string) segment {
	if strings.HasPrefix(part, "*") {
		return segment{kind: segWildcard, name: part[1:]}
	}
	fmtCap := false
	if strings.HasSuffix(part, ".:format") {
		part = strings.TrimSuffix(part, ".:format")
		fmtCap = true
	}
	if strings.HasPrefix(part, ":") {
		return segment{kind: segParam, name: part[1:], fmtCap: fmtCap}
	}
	return segment{kind: segLiteral, lit: part, fmtCap: fmtCap}
}

// matchPath attempts to match a request path against the route's segments. On
// success it returns the captured path params (including "format" when a
// ".:format" suffix or a bare ".ext" on a literal was present) and true.
func (r *Route) matchPath(path string) (map[string]string, bool) {
	reqParts := splitPath(path)
	params := map[string]string{}
	si, pi := 0, 0
	for si < len(r.segs) {
		seg := r.segs[si]
		if seg.kind == segWildcard {
			// A wildcard consumes every remaining component, joined by "/".
			rest := strings.Join(reqParts[pi:], "/")
			params[seg.name] = rest
			return params, true
		}
		if pi >= len(reqParts) {
			return nil, false
		}
		comp := reqParts[pi]
		switch seg.kind {
		case segLiteral:
			if seg.fmtCap {
				base, format, ok := splitFormat(comp)
				if !ok || base != seg.lit {
					return nil, false
				}
				params["format"] = format
			} else if comp != seg.lit {
				return nil, false
			}
		case segParam:
			val := comp
			if seg.fmtCap {
				base, format, ok := splitFormat(comp)
				if !ok {
					return nil, false
				}
				val, params["format"] = base, format
			}
			params[seg.name] = val
		}
		si++
		pi++
	}
	if pi != len(reqParts) {
		return nil, false
	}
	return params, true
}

// splitPath trims surrounding slashes and splits into components; an empty path
// yields no components (so "/" matches a zero-segment route).
func splitPath(path string) []string {
	p := strings.Trim(path, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// splitFormat separates a "base.ext" component into its base and extension. It
// splits on the final dot; a component with no dot (or a trailing/leading dot)
// is not a valid ".:format" capture.
func splitFormat(comp string) (base, format string, ok bool) {
	i := strings.LastIndex(comp, ".")
	if i <= 0 || i == len(comp)-1 {
		return "", "", false
	}
	return comp[:i], comp[i+1:], true
}

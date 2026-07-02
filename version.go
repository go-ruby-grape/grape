// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"regexp"
	"strings"
)

// VersionStrategy names how Grape locates the requested API version.
type VersionStrategy int

const (
	// VersionPath: the version is the first path segment ("/v1/…").
	VersionPath VersionStrategy = iota
	// VersionHeader: the version is read from an Accept header vendor tree,
	// e.g. "Accept: application/vnd.vendor-v1+json".
	VersionHeader
	// VersionParam: the version is a query/body parameter (default name "apiver").
	VersionParam
	// VersionAccept: the version is the whole Accept media-type parameter
	// ("Accept: application/json; version=v1"), less common than VersionHeader.
	VersionAccept
)

// Version describes an API's versioning configuration.
type Version struct {
	Name     string // the version string, e.g. "v1"
	Strategy VersionStrategy
	Vendor   string // vendor for VersionHeader (application/vnd.<vendor>-<name>+<fmt>)
	Param    string // parameter name for VersionParam (default "apiver")
}

// Prefix returns the path prefix a route acquires under this version, i.e. the
// version segment for VersionPath and the empty string otherwise (the version
// then lives in a header or param, not the path).
func (v *Version) Prefix() string {
	if v == nil || v.Strategy != VersionPath {
		return ""
	}
	return "/" + strings.Trim(v.Name, "/")
}

// ParamName returns the effective parameter name for VersionParam.
func (v *Version) ParamName() string {
	if v.Param != "" {
		return v.Param
	}
	return "apiver"
}

var vendorHeaderRe = regexp.MustCompile(`\bvnd\.([^.+;\s-]+)-([^.+;\s]+)`)

// Matches reports whether a request carries this version. path is the raw request
// path (only consulted for VersionPath), accept is the Accept header, and param
// is the version parameter value (only consulted for VersionParam / when set).
// For VersionPath the caller strips the matched prefix separately via Prefix.
func (v *Version) Matches(path, accept, param string) bool {
	switch v.Strategy {
	case VersionPath:
		seg := firstSegment(path)
		return seg == strings.Trim(v.Name, "/")
	case VersionHeader:
		m := vendorHeaderRe.FindStringSubmatch(accept)
		if m == nil {
			return false
		}
		if v.Vendor != "" && m[1] != v.Vendor {
			return false
		}
		return m[2] == v.Name
	case VersionAccept:
		return acceptVersionParam(accept) == v.Name
	case VersionParam:
		return param == v.Name
	}
	return false
}

func firstSegment(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// acceptVersionParam extracts a "version=" media-type parameter from an Accept
// header, e.g. "application/json; version=v2" → "v2".
func acceptVersionParam(accept string) string {
	for _, part := range strings.Split(accept, ";") {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 && strings.TrimSpace(kv[0]) == "version" {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

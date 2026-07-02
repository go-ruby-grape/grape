// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "strings"

// contentTypes maps Grape's built-in format symbols to their default MIME types,
// in the order Grape registers them (json first — it is the default when the
// client expresses no preference and no format is forced).
var contentTypes = []struct {
	format string
	mime   string
}{
	{"json", "application/json"},
	{"jsonapi", "application/vnd.api+json"},
	{"xml", "application/xml"},
	{"serializable_hash", "application/json"},
	{"txt", "text/plain"},
	{"binary", "application/octet-stream"},
}

// MimeFor returns the MIME type Grape emits for a format symbol, or "" if the
// format is unknown.
func MimeFor(format string) string {
	for _, ct := range contentTypes {
		if ct.format == format {
			return ct.mime
		}
	}
	return ""
}

// Negotiate resolves the response format for a request. It applies Grape's
// precedence: an explicit ".:format" path extension (extFormat, may be "") wins;
// otherwise a forced API-wide format (forced, may be "") wins; otherwise the
// Accept header is matched against the API's offered formats. offered lists the
// format symbols the API serves, in preference order.
//
// It returns the chosen format and true, or ("", false) when the request cannot
// be satisfied — the 406 Not Acceptable decision.
func Negotiate(extFormat, forced, accept string, offered []string) (string, bool) {
	// 1. Explicit extension: it must be one the API offers.
	if extFormat != "" {
		if contains(offered, extFormat) {
			return extFormat, true
		}
		return "", false
	}
	// 2. Forced format: content negotiation is bypassed entirely.
	if forced != "" {
		return forced, true
	}
	// 3. No Accept (or */*): the first offered format (json by convention).
	accept = strings.TrimSpace(accept)
	if accept == "" || acceptsAny(accept) {
		if len(offered) == 0 {
			return "", false
		}
		return offered[0], true
	}
	// 4. Match each acceptable MIME against the offered formats.
	for _, mime := range parseAccept(accept) {
		for _, f := range offered {
			if MimeFor(f) == mime {
				return f, true
			}
		}
	}
	return "", false
}

// parseAccept splits an Accept header into its bare media types, in header order
// (quality ordering is not modelled — Grape iterates its offered formats). Each
// entry is lower-cased with parameters stripped.
func parseAccept(accept string) []string {
	var out []string
	for _, part := range strings.Split(accept, ",") {
		mt := strings.TrimSpace(part)
		if i := strings.IndexByte(mt, ';'); i >= 0 {
			mt = strings.TrimSpace(mt[:i])
		}
		if mt != "" {
			out = append(out, strings.ToLower(mt))
		}
	}
	return out
}

// acceptsAny reports whether the Accept header contains a "*/*" wildcard.
func acceptsAny(accept string) bool {
	for _, mt := range parseAccept(accept) {
		if mt == "*/*" {
			return true
		}
	}
	return false
}

func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}

// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "testing"

func TestMimeFor(t *testing.T) {
	if MimeFor("json") != "application/json" {
		t.Fatal("json mime")
	}
	if MimeFor("txt") != "text/plain" {
		t.Fatal("txt mime")
	}
	if MimeFor("xml") != "application/xml" {
		t.Fatal("xml mime")
	}
	if MimeFor("nope") != "" {
		t.Fatal("unknown mime should be empty")
	}
}

func TestNegotiate(t *testing.T) {
	offered := []string{"json", "xml", "txt"}
	cases := []struct {
		ext, forced, accept string
		want                string
		ok                  bool
	}{
		// Explicit extension wins when offered.
		{"xml", "", "application/json", "xml", true},
		// Extension not offered -> 406.
		{"yaml", "", "", "", false},
		// Forced format bypasses negotiation.
		{"", "json", "application/xml", "json", true},
		// No Accept -> first offered.
		{"", "", "", "json", true},
		// */* -> first offered.
		{"", "", "*/*", "json", true},
		// Accept matches an offered MIME.
		{"", "", "application/xml", "xml", true},
		{"", "", "text/plain", "txt", true},
		// Accept with params + multiple types, first match wins.
		{"", "", "application/xml; q=0.9, text/plain", "xml", true},
		// Unsatisfiable Accept -> 406.
		{"", "", "image/png", "", false},
	}
	for _, c := range cases {
		got, ok := Negotiate(c.ext, c.forced, c.accept, offered)
		if got != c.want || ok != c.ok {
			t.Errorf("Negotiate(%q,%q,%q) = (%q,%v) want (%q,%v)", c.ext, c.forced, c.accept, got, ok, c.want, c.ok)
		}
	}
}

func TestNegotiateEmptyOffered(t *testing.T) {
	if _, ok := Negotiate("", "", "", nil); ok {
		t.Fatal("no offered formats and no Accept -> 406")
	}
}

func TestParseAcceptAndAny(t *testing.T) {
	got := parseAccept("application/json ; q=1, , text/plain")
	if len(got) != 2 || got[0] != "application/json" || got[1] != "text/plain" {
		t.Fatalf("parseAccept = %v", got)
	}
	if !acceptsAny("text/html, */*") {
		t.Fatal("should detect */*")
	}
	if acceptsAny("application/json") {
		t.Fatal("no wildcard present")
	}
}

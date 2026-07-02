// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "testing"

func TestVersionPrefix(t *testing.T) {
	v := &Version{Name: "v1", Strategy: VersionPath}
	if v.Prefix() != "/v1" {
		t.Fatalf("prefix = %q", v.Prefix())
	}
	// Non-path strategies carry no path prefix.
	if (&Version{Name: "v1", Strategy: VersionHeader}).Prefix() != "" {
		t.Fatal("header strategy should have no prefix")
	}
	var nilV *Version
	if nilV.Prefix() != "" {
		t.Fatal("nil version prefix should be empty")
	}
}

func TestVersionParamName(t *testing.T) {
	if (&Version{Strategy: VersionParam}).ParamName() != "apiver" {
		t.Fatal("default param name")
	}
	if (&Version{Strategy: VersionParam, Param: "v"}).ParamName() != "v" {
		t.Fatal("custom param name")
	}
}

func TestVersionMatches(t *testing.T) {
	path := &Version{Name: "v1", Strategy: VersionPath}
	if !path.Matches("/v1/cats", "", "") {
		t.Fatal("path v1 should match")
	}
	if path.Matches("/v2/cats", "", "") {
		t.Fatal("path v2 should not match v1")
	}
	if path.Matches("/", "", "") {
		t.Fatal("empty path should not match")
	}

	hdr := &Version{Name: "v2", Strategy: VersionHeader, Vendor: "acme"}
	if !hdr.Matches("", "application/vnd.acme-v2+json", "") {
		t.Fatal("vendor header v2 should match")
	}
	if hdr.Matches("", "application/vnd.acme-v3+json", "") {
		t.Fatal("v3 should not match v2")
	}
	if hdr.Matches("", "application/vnd.other-v2+json", "") {
		t.Fatal("wrong vendor should not match")
	}
	if hdr.Matches("", "application/json", "") {
		t.Fatal("no vendor tree should not match")
	}
	// Vendor unset: any vendor accepted.
	anyVendor := &Version{Name: "v2", Strategy: VersionHeader}
	if !anyVendor.Matches("", "application/vnd.whatever-v2+json", "") {
		t.Fatal("empty vendor should accept any")
	}

	acc := &Version{Name: "v1", Strategy: VersionAccept}
	if !acc.Matches("", "application/json; version=v1", "") {
		t.Fatal("accept-param version should match")
	}
	if acc.Matches("", "application/json", "") {
		t.Fatal("no version param should not match")
	}

	param := &Version{Name: "v1", Strategy: VersionParam}
	if !param.Matches("", "", "v1") {
		t.Fatal("param version should match")
	}
	if param.Matches("", "", "v2") {
		t.Fatal("param v2 should not match v1")
	}

	// An unknown strategy never matches.
	unknown := &Version{Name: "v1", Strategy: VersionStrategy(99)}
	if unknown.Matches("/v1", "", "v1") {
		t.Fatal("unknown strategy should not match")
	}
}

func TestAcceptVersionParam(t *testing.T) {
	if got := acceptVersionParam("application/json; charset=utf-8; version=v3"); got != "v3" {
		t.Fatalf("got %q", got)
	}
	if got := acceptVersionParam("application/json"); got != "" {
		t.Fatalf("got %q", got)
	}
}

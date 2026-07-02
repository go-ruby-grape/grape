// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "testing"

func TestMutuallyExclusive(t *testing.T) {
	set := &ParamSet{
		Params:            []*Param{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		MutuallyExclusive: [][]string{{"a", "b", "c"}},
	}
	if _, e := errString(t, set, Raw{"a": "1"}); e != "" {
		t.Fatalf("one present should pass: %q", e)
	}
	if _, e := errString(t, set, Raw{"a": "1", "b": "2", "c": "3"}); e != "a, b, c are mutually exclusive" {
		t.Fatalf("got %q", e)
	}
	// Blank values still count as present; only the present names are reported.
	if _, e := errString(t, set, Raw{"a": "", "b": ""}); e != "a, b are mutually exclusive" {
		t.Fatalf("got %q", e)
	}
	// A non-contiguous subset reports just those present.
	if _, e := errString(t, set, Raw{"a": "1", "c": "3"}); e != "a, c are mutually exclusive" {
		t.Fatalf("got %q", e)
	}
}

func TestExactlyOneOf(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, ExactlyOneOf: [][]string{{"a", "b"}}}
	if _, e := errString(t, set, Raw{}); e != "a, b are missing, exactly one parameter must be provided" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"a": "1", "b": "2"}); e != "a, b are mutually exclusive" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"a": "1"}); e != "" {
		t.Fatalf("got %q", e)
	}
}

func TestAtLeastOneOf(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, AtLeastOneOf: [][]string{{"a", "b"}}}
	if _, e := errString(t, set, Raw{}); e != "a, b are missing, at least one parameter must be provided" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"a": ""}); e != "" {
		t.Fatalf("blank present should satisfy: %q", e)
	}
}

func TestAllOrNoneOf(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, AllOrNoneOf: [][]string{{"a", "b"}}}
	if _, e := errString(t, set, Raw{"a": "1"}); e != "a, b provide all or none of parameters" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{}); e != "" {
		t.Fatalf("none present should pass: %q", e)
	}
	if _, e := errString(t, set, Raw{"a": "1", "b": "2"}); e != "" {
		t.Fatalf("all present should pass: %q", e)
	}
}

func TestNestedExclusivity(t *testing.T) {
	set := &ParamSet{Params: []*Param{{
		Name: "grp", Required: true, IsHash: true,
		Group: &ParamSet{
			Params:            []*Param{{Name: "a"}, {Name: "b"}},
			MutuallyExclusive: [][]string{{"a", "b"}},
		},
	}}}
	if _, e := errString(t, set, Raw{"grp": map[string]any{"a": "1", "b": "2"}}); e != "grp[a], grp[b] are mutually exclusive" {
		t.Fatalf("got %q", e)
	}
}

func TestNilGroupHash(t *testing.T) {
	// A Hash param with no group declaration yields an empty coerced sub-map and
	// no missing-inner report even when required.
	set := &ParamSet{Params: []*Param{{Name: "grp", Required: true, IsHash: true}}}
	c, e := errString(t, set, Raw{"grp": map[string]any{"x": "1"}})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	if sub, ok := c["grp"].(map[string]any); !ok || len(sub) != 0 {
		t.Fatalf("grp = %#v", c["grp"])
	}
	// Missing required Hash without a group: only "grp is missing".
	if _, e := errString(t, set, Raw{}); e != "grp is missing" {
		t.Fatalf("got %q", e)
	}
}

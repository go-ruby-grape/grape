// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"math/big"
	"reflect"
	"testing"
	"time"
)

// errString runs Validate and returns the joined error string ("" on success).
func errString(t *testing.T, set *ParamSet, raw Raw) (map[string]any, string) {
	t.Helper()
	c, errs := NewParamsValidator(set).Validate(raw)
	if errs == nil {
		return c, ""
	}
	return c, errs.Error()
}

func TestValidatePresenceAndCoercion(t *testing.T) {
	set := &ParamSet{Params: []*Param{
		{Name: "id", Required: true, Type: TypeInteger},
		{Name: "name", Type: TypeString, Values: []any{"a", "b", "c"}},
		{Name: "score", Type: TypeFloat},
		{Name: "active", Type: TypeBoolean},
	}}
	// Valid: id coerced to int64, others absent.
	c, e := errString(t, set, Raw{"id": "42"})
	if e != "" {
		t.Fatalf("unexpected error %q", e)
	}
	if c["id"] != int64(42) {
		t.Fatalf("id = %#v", c["id"])
	}
	// Invalid integer.
	if _, e := errString(t, set, Raw{"id": "x"}); e != "id is invalid" {
		t.Fatalf("got %q", e)
	}
	// Missing required.
	if _, e := errString(t, set, Raw{}); e != "id is missing" {
		t.Fatalf("got %q", e)
	}
	// Values violation.
	if _, e := errString(t, set, Raw{"id": "42", "name": "z"}); e != "name does not have a valid value" {
		t.Fatalf("got %q", e)
	}
	// Valid value + float + boolean.
	c, e = errString(t, set, Raw{"id": "42", "name": "a", "score": "3.14", "active": "true"})
	if e != "" || c["score"] != 3.14 || c["active"] != true {
		t.Fatalf("c=%#v e=%q", c, e)
	}
	// Invalid float + invalid boolean.
	if _, e := errString(t, set, Raw{"id": "42", "score": "abc"}); e != "score is invalid" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"id": "42", "active": "nope"}); e != "active is invalid" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateMultipleMissing(t *testing.T) {
	set := &ParamSet{Params: []*Param{
		{Name: "a", Required: true, Type: TypeInteger},
		{Name: "b", Required: true, Type: TypeInteger},
	}}
	if _, e := errString(t, set, Raw{}); e != "a is missing, b is missing" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateLength(t *testing.T) {
	within := &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, HasMinLen: true, MinLen: 2, HasMaxLen: true, MaxLen: 5}}}
	if _, e := errString(t, within, Raw{"name": "x"}); e != "name is expected to have length within 2 and 5" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, within, Raw{"name": "toolong"}); e != "name is expected to have length within 2 and 5" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, within, Raw{"name": "ab"}); e != "" {
		t.Fatalf("got %q", e)
	}
	minOnly := &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, HasMinLen: true, MinLen: 3}}}
	if _, e := errString(t, minOnly, Raw{"name": "ab"}); e != "name is expected to have length greater than or equal to 3" {
		t.Fatalf("got %q", e)
	}
	maxOnly := &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, HasMaxLen: true, MaxLen: 3}}}
	if _, e := errString(t, maxOnly, Raw{"name": "abcd"}); e != "name is expected to have length less than or equal to 3" {
		t.Fatalf("got %q", e)
	}
	// length on a non-string/array value is a no-op.
	numLen := &ParamSet{Params: []*Param{{Name: "n", Required: true, Type: TypeInteger, HasMinLen: true, MinLen: 3}}}
	if _, e := errString(t, numLen, Raw{"n": "7"}); e != "" {
		t.Fatalf("length on int should be skipped, got %q", e)
	}
}

func TestValidateRegexpAndExcept(t *testing.T) {
	re := &ParamSet{Params: []*Param{{Name: "email", Required: true, Regexp: Regexp{Match: func(s string) bool {
		for i := 0; i < len(s); i++ {
			if s[i] == '@' && i > 0 && i < len(s)-1 {
				return true
			}
		}
		return false
	}}}}}
	if _, e := errString(t, re, Raw{"email": "bad"}); e != "email is invalid" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, re, Raw{"email": "a@b"}); e != "" {
		t.Fatalf("got %q", e)
	}
	except := &ParamSet{Params: []*Param{{Name: "x", Type: TypeInteger, ExceptValues: []any{int64(1), int64(2)}}}}
	if _, e := errString(t, except, Raw{"x": "1"}); e != "x has a value not allowed" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, except, Raw{"x": "3"}); e != "" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateRegexpNonString(t *testing.T) {
	// A regexp on a coerced non-string value stringifies it before matching.
	set := &ParamSet{Params: []*Param{{Name: "n", Type: TypeInteger, Regexp: Regexp{Match: func(s string) bool { return s == "42" }}}}}
	if _, e := errString(t, set, Raw{"n": "42"}); e != "" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"n": "7"}); e != "n is invalid" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateRange(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "n", Required: true, Type: TypeInteger, HasRange: true, RangeMin: 1, RangeMax: 10}}}
	if _, e := errString(t, set, Raw{"n": "5"}); e != "" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"n": "55"}); e != "n does not have a valid value" {
		t.Fatalf("got %q", e)
	}
	// Range against a non-integer coerced value fails the range check.
	set2 := &ParamSet{Params: []*Param{{Name: "n", Type: TypeString, HasRange: true, RangeMin: 1, RangeMax: 10}}}
	if _, e := errString(t, set2, Raw{"n": "x"}); e != "n does not have a valid value" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateAllowBlank(t *testing.T) {
	f := false
	set := &ParamSet{Params: []*Param{{Name: "nm", Required: true, Type: TypeString, AllowBlank: &f}}}
	if _, e := errString(t, set, Raw{"nm": ""}); e != "nm is empty" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"nm": "  "}); e != "nm is empty" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"nm": "ok"}); e != "" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateDefaults(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "d", HasDefault: true, Default: int64(7)}}}
	c, e := errString(t, set, Raw{})
	if e != "" || c["d"] != int64(7) {
		t.Fatalf("c=%#v e=%q", c, e)
	}
	// Callable default.
	setf := &ParamSet{Params: []*Param{{Name: "d", DefaultFunc: func() any { return int64(99) }}}}
	c, _ = errString(t, setf, Raw{})
	if c["d"] != int64(99) {
		t.Fatalf("d=%#v", c["d"])
	}
	// Present value overrides default.
	c, _ = errString(t, set, Raw{"d": "3"})
	if c["d"] != "3" {
		t.Fatalf("present should override default: %#v", c["d"])
	}
}

func TestValidateCoerceWith(t *testing.T) {
	up := &ParamSet{Params: []*Param{{Name: "up", Type: TypeString, CoerceWith: func(raw string) (any, bool) {
		return "HI", raw != ""
	}}}}
	c, e := errString(t, up, Raw{"up": "hi"})
	if e != "" || c["up"] != "HI" {
		t.Fatalf("c=%#v e=%q", c, e)
	}
	// coerce_with returning false is "is invalid".
	if _, e := errString(t, up, Raw{"up": ""}); e != "up is invalid" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateBlankOptionalScalar(t *testing.T) {
	// A blank optional integer coerces to nil and is skipped for value checks.
	set := &ParamSet{Params: []*Param{{Name: "x", Type: TypeInteger, Values: []any{int64(9)}}}}
	c, e := errString(t, set, Raw{"x": ""})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	if v, ok := c["x"]; !ok || v != nil {
		t.Fatalf("x = %#v (present=%v), want nil present", v, ok)
	}
}

func TestValidateNonStringLeaf(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "x", Type: TypeInteger}}}
	// A nested map where a scalar is expected is invalid.
	if _, e := errString(t, set, Raw{"x": map[string]any{"a": "b"}}); e != "x is invalid" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateArray(t *testing.T) {
	strs := &ParamSet{Params: []*Param{{Name: "tags", Required: true, IsArray: true, ElemType: TypeString}}}
	c, e := errString(t, strs, Raw{"tags": []any{"x", "y"}})
	if e != "" || !reflect.DeepEqual(c["tags"], []any{"x", "y"}) {
		t.Fatalf("c=%#v e=%q", c, e)
	}
	ints := &ParamSet{Params: []*Param{{Name: "n", Required: true, IsArray: true, ElemType: TypeInteger}}}
	c, e = errString(t, ints, Raw{"n": []any{"1", "2"}})
	if e != "" || !reflect.DeepEqual(c["n"], []any{int64(1), int64(2)}) {
		t.Fatalf("c=%#v e=%q", c, e)
	}
	// Bad element.
	if _, e := errString(t, ints, Raw{"n": []any{"1", "z"}}); e != "n is invalid" {
		t.Fatalf("got %q", e)
	}
	// Non-array raw value.
	if _, e := errString(t, ints, Raw{"n": "notarray"}); e != "n is invalid" {
		t.Fatalf("got %q", e)
	}
	// Non-string element (e.g. a nested map) in an array.
	if _, e := errString(t, strs, Raw{"tags": []any{map[string]any{}}}); e != "tags is invalid" {
		t.Fatalf("got %q", e)
	}
	// Bare Array (no element type) passes strings through.
	bare := &ParamSet{Params: []*Param{{Name: "a", IsArray: true}}}
	c, _ = errString(t, bare, Raw{"a": []any{"z"}})
	if !reflect.DeepEqual(c["a"], []any{"z"}) {
		t.Fatalf("bare array = %#v", c["a"])
	}
}

func TestValidateNestedHash(t *testing.T) {
	set := &ParamSet{Params: []*Param{{
		Name: "grp", Required: true, IsHash: true,
		Group: &ParamSet{Params: []*Param{{Name: "inner", Required: true, Type: TypeInteger}}},
	}}}
	// Valid nested.
	c, e := errString(t, set, Raw{"grp": map[string]any{"inner": "5"}})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	sub := c["grp"].(map[string]any)
	if sub["inner"] != int64(5) {
		t.Fatalf("inner = %#v", sub["inner"])
	}
	// Bad inner coercion nests the name.
	if _, e := errString(t, set, Raw{"grp": map[string]any{"inner": "x"}}); e != "grp[inner] is invalid" {
		t.Fatalf("got %q", e)
	}
	// Missing inner: group present but inner absent.
	if _, e := errString(t, set, Raw{"grp": map[string]any{}}); e != "grp[inner] is missing" {
		t.Fatalf("got %q", e)
	}
	// Missing whole group: reports group + inner.
	if _, e := errString(t, set, Raw{}); e != "grp is missing, grp[inner] is missing" {
		t.Fatalf("got %q", e)
	}
	// Non-map value for a Hash param.
	if _, e := errString(t, set, Raw{"grp": "x"}); e != "grp is invalid" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateOptionalHashAbsent(t *testing.T) {
	// An absent optional Hash simply produces nothing (no missing-group report).
	set := &ParamSet{Params: []*Param{{Name: "grp", IsHash: true, Group: &ParamSet{Params: []*Param{{Name: "inner", Required: true, Type: TypeInteger}}}}}}
	c, e := errString(t, set, Raw{})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	if _, ok := c["grp"]; ok {
		t.Fatalf("absent optional hash should not appear: %#v", c)
	}
}

func TestValidateBigIntValues(t *testing.T) {
	big1, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	set := &ParamSet{Params: []*Param{{Name: "n", Type: TypeInteger, Values: []any{big1}}}}
	c, e := errString(t, set, Raw{"n": "123456789012345678901234567890"})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	if got, ok := c["n"].(*big.Int); !ok || got.Cmp(big1) != 0 {
		t.Fatalf("n = %#v", c["n"])
	}
	// A mismatching big value is rejected.
	if _, e := errString(t, set, Raw{"n": "999999999999999999999999999999"}); e != "n does not have a valid value" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateDateTime(t *testing.T) {
	d := &ParamSet{Params: []*Param{{Name: "d", Required: true, Type: TypeDate}}}
	c, e := errString(t, d, Raw{"d": "2026-07-02"})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	if tm := c["d"].(time.Time); tm.Year() != 2026 || tm.Month() != 7 || tm.Day() != 2 {
		t.Fatalf("d = %v", tm)
	}
	if _, e := errString(t, d, Raw{"d": "notadate"}); e != "d is invalid" {
		t.Fatalf("got %q", e)
	}
	tt := &ParamSet{Params: []*Param{{Name: "t", Required: true, Type: TypeTime}}}
	if _, e := errString(t, tt, Raw{"t": "2026-07-02T15:04:05Z"}); e != "" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, tt, Raw{"t": "bad"}); e != "t is invalid" {
		t.Fatalf("got %q", e)
	}
}

func TestValidateJSONAndFile(t *testing.T) {
	j := &ParamSet{Params: []*Param{{Name: "j", Required: true, Type: TypeJSON}}}
	c, e := errString(t, j, Raw{"j": `{"a":1}`})
	if e != "" {
		t.Fatalf("got %q", e)
	}
	if m, ok := c["j"].(map[string]any); !ok || m["a"] != float64(1) {
		t.Fatalf("j = %#v", c["j"])
	}
	if _, e := errString(t, j, Raw{"j": "notjson"}); e != "j is invalid" {
		t.Fatalf("got %q", e)
	}
	f := &ParamSet{Params: []*Param{{Name: "f", Type: TypeFile}}}
	c, _ = errString(t, f, Raw{"f": "handle"})
	if c["f"] != "handle" {
		t.Fatalf("file = %#v", c["f"])
	}
}

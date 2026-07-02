// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Type names the coercion targets Grape's params DSL understands. A scalar type
// coerces a single raw string; Array[T] / Hash coerce collections (handled in
// validate.go via ElemType).
type Type string

const (
	TypeString  Type = "String"
	TypeInteger Type = "Integer"
	TypeFloat   Type = "Float"
	TypeBoolean Type = "Boolean"
	TypeDate    Type = "Date"
	TypeTime    Type = "Time"
	TypeJSON    Type = "JSON"
	TypeArray   Type = "Array"
	TypeHash    Type = "Hash"
	TypeFile    Type = "File"
)

// coerceScalar coerces a single raw string to the given scalar type. It returns
// the coerced Go value and true, or (nil, false) when the value is invalid — the
// caller then raises "<param> is invalid". An empty string coerces to nil for
// every non-string scalar type (Grape treats a blank optional scalar as absent),
// while for String it stays "".
func coerceScalar(t Type, raw string) (any, bool) {
	if raw == "" && t != TypeString {
		return nil, true
	}
	switch t {
	case TypeString, "":
		return raw, true
	case TypeInteger:
		return coerceInteger(raw)
	case TypeFloat:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, false
		}
		return f, true
	case TypeBoolean:
		return coerceBoolean(raw)
	case TypeDate:
		tm, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return nil, false
		}
		return tm, true
	case TypeTime:
		return coerceTime(raw)
	case TypeJSON:
		return coerceJSON(raw)
	case TypeFile:
		// A File param is an upload stub the host fills; the deterministic core
		// only records the raw handle.
		return raw, true
	}
	return nil, false
}

// coerceInteger parses a base-10 integer, promoting to *big.Int on overflow to
// mirror Ruby's unbounded Integer. A decimal point or any non-integer text fails.
func coerceInteger(raw string) (any, bool) {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err == nil {
		return n, true
	}
	if bi, ok := new(big.Int).SetString(raw, 10); ok {
		return bi, true
	}
	return nil, false
}

// coerceBoolean maps Grape's accepted truthy/falsey tokens. Grape (via
// Virtus/dry-types) accepts "true"/"1"/"t"/"yes"/"on" and their negatives,
// case-insensitively; anything else is invalid.
func coerceBoolean(raw string) (any, bool) {
	switch strings.ToLower(raw) {
	case "true", "1", "t", "yes", "y", "on":
		return true, true
	case "false", "0", "f", "no", "n", "off":
		return false, true
	}
	return nil, false
}

// coerceTime parses an ISO-8601 / RFC3339 timestamp, then a date-only form.
func coerceTime(raw string) (any, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if tm, err := time.Parse(layout, raw); err == nil {
			return tm, true
		}
	}
	return nil, false
}

// coerceJSON parses a JSON document into Go's generic model (map/slice/…), the
// shape Grape yields for a JSON-typed param.
func coerceJSON(raw string) (any, bool) {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, false
	}
	return v, true
}

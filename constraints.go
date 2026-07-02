// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"fmt"
	"math/big"
	"strings"
)

// checkValues enforces values / except_values, returning Grape's message
// fragment or "" when the value is allowed. An explicit value set uses "does not
// have a valid value"; an integer range uses the same; except_values uses "has a
// value not allowed".
func checkValues(p *Param, val any) string {
	if p.HasRange {
		if n, ok := toInt64(val); !ok || n < p.RangeMin || n > p.RangeMax {
			return "does not have a valid value"
		}
	}
	if len(p.Values) > 0 && !valueIn(p.Values, val) {
		return "does not have a valid value"
	}
	if len(p.ExceptValues) > 0 && valueIn(p.ExceptValues, val) {
		return "has a value not allowed"
	}
	return ""
}

// checkLength enforces the length: constraint on a String or Array value. Grape
// phrases the message by which bounds are present: "within M and N", "greater
// than or equal to M", or "less than or equal to N".
func checkLength(p *Param, val any) string {
	if !p.HasMinLen && !p.HasMaxLen {
		return ""
	}
	n, ok := lengthOf(val)
	if !ok {
		return ""
	}
	switch {
	case p.HasMinLen && p.HasMaxLen:
		if n < p.MinLen || n > p.MaxLen {
			return fmt.Sprintf("is expected to have length within %d and %d", p.MinLen, p.MaxLen)
		}
	case p.HasMinLen:
		if n < p.MinLen {
			return fmt.Sprintf("is expected to have length greater than or equal to %d", p.MinLen)
		}
	case p.HasMaxLen:
		if n > p.MaxLen {
			return fmt.Sprintf("is expected to have length less than or equal to %d", p.MaxLen)
		}
	}
	return ""
}

// checkRegexp enforces a regexp: constraint against a value's string form,
// returning "is invalid" on mismatch (Grape's regexp failure message).
func checkRegexp(p *Param, val any) string {
	if p.Regexp.Match == nil {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		s = fmt.Sprint(val)
	}
	if !p.Regexp.Match(s) {
		return "is invalid"
	}
	return ""
}

// lengthOf reports the length of a String (rune count) or Array value.
func lengthOf(val any) (int, bool) {
	switch v := val.(type) {
	case string:
		return len([]rune(v)), true
	case []any:
		return len(v), true
	}
	return 0, false
}

// valueIn reports whether val equals any member of set, comparing integers
// across int64/*big.Int and everything else by value.
func valueIn(set []any, val any) bool {
	for _, s := range set {
		if valuesEqual(s, val) {
			return true
		}
	}
	return false
}

// valuesEqual compares two coerced values, bridging int64 and *big.Int so a
// value list written as ints matches a big-int-coerced param and vice versa.
func valuesEqual(a, b any) bool {
	if an, aok := toBigInt(a); aok {
		if bn, bok := toBigInt(b); bok {
			return an.Cmp(bn) == 0
		}
		return false
	}
	return a == b
}

// toInt64 narrows an integer value (int/int64/*big.Int within range) to int64.
func toInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case *big.Int:
		if v.IsInt64() {
			return v.Int64(), true
		}
	}
	return 0, false
}

// toBigInt promotes an integer value to *big.Int for cross-width comparison.
func toBigInt(val any) (*big.Int, bool) {
	switch v := val.(type) {
	case int:
		return big.NewInt(int64(v)), true
	case int64:
		return big.NewInt(v), true
	case *big.Int:
		return v, true
	}
	return nil, false
}

// joinNames renders a parameter-name list the way Grape does in exclusivity
// messages: comma-space separated ("a, b").
func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

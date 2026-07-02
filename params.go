// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

// Param is one declared parameter in a `params do … end` block. It mirrors the
// options a `requires`/`optional` line carries. The host builds these from the
// Ruby DSL; the validator consumes them.
type Param struct {
	Name     string
	Required bool // requires vs optional

	Type     Type // scalar type; "" means untyped (raw string passthrough)
	ElemType Type // element type for Array[T]; "" for a bare Array
	IsArray  bool // type: Array or Array[T]
	IsHash   bool // type: Hash (a nested group container)

	// Values / ExceptValues constrain the coerced value. A value list may be an
	// explicit set (Values) or an inclusive integer range (RangeMin..RangeMax
	// when HasRange is set).
	Values       []any
	ExceptValues []any
	HasRange     bool
	RangeMin     int64
	RangeMax     int64

	// Length constraints (String/Array). Present flags gate each bound.
	HasMinLen bool
	MinLen    int
	HasMaxLen bool
	MaxLen    int

	Regexp     Regexp // regexp: /…/ — matched via the injected matcher
	AllowBlank *bool  // allow_blank: false rejects "" / whitespace; nil = default true

	// Default supplies a value when the param is absent. If DefaultFunc is set it
	// is called lazily (callable default); otherwise Default is used verbatim.
	HasDefault  bool
	Default     any
	DefaultFunc func() any

	// CoerceWith replaces built-in coercion with a host lambda; it receives the
	// raw string and returns the coerced value (and false to signal invalid).
	CoerceWith func(raw string) (any, bool)

	// Group holds the nested declarations of a Hash param (`requires :g, type:
	// Hash do … end`), including its own cross-parameter validators, so nested
	// exclusivity reports names as "grp[a], grp[b]".
	Group *ParamSet
}

// Regexp is a compiled-pattern reference. The deterministic core does not embed
// a regexp engine (that is go-ruby-regexp's job); the host supplies a Match
// function. A zero Regexp (Match nil) means "no regexp constraint".
type Regexp struct {
	Source string              // the pattern source, for diagnostics
	Match  func(s string) bool // returns true when s matches
}

// ParamSet is a full `params do … end` declaration plus the cross-parameter
// group validators (mutually_exclusive and friends).
type ParamSet struct {
	Params []*Param

	MutuallyExclusive [][]string
	ExactlyOneOf      [][]string
	AtLeastOneOf      [][]string
	AllOrNoneOf       [][]string
}

// scalarType reports the type used to coerce a leaf (non-array, non-hash) value:
// the declared type, or "" for an untyped String-passthrough. Array elements are
// coerced against ElemType directly in coerceArray, so this is leaf-only.
func (p *Param) scalarType() Type {
	return p.Type
}

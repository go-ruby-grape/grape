// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"fmt"
	"strings"
)

// ParamsValidator validates and coerces raw request params against a ParamSet.
type ParamsValidator struct {
	set *ParamSet
}

// NewParamsValidator builds a validator for a declaration.
func NewParamsValidator(set *ParamSet) *ParamsValidator {
	return &ParamsValidator{set: set}
}

// Raw is the request parameter tree the host hands in. A leaf is a string; a
// nested Hash param is a map[string]any; an array is a []any of strings (or of
// nested maps for Array-of-Hash). This mirrors what Rack parses from a query /
// form / JSON body.
type Raw = map[string]any

// Validate coerces raw against the declaration. On success it returns the coerced
// params map and nil. On failure it returns the partially-coerced map and a
// *ValidationErrors carrying every failure in declaration order — the same order
// (and messages) Grape renders into its 400 body.
func (v *ParamsValidator) Validate(raw Raw) (map[string]any, *ValidationErrors) {
	errs := &ValidationErrors{}
	coerced := validateGroup(v.set.Params, raw, "", errs)
	validateExclusivity(v.set, raw, "", errs)
	if errs.Empty() {
		return coerced, nil
	}
	return coerced, errs
}

// validateGroup coerces one level of parameters (top-level or a nested Hash),
// prefixing nested parameter names with prefix ("grp" → "grp[inner]").
func validateGroup(params []*Param, raw Raw, prefix string, errs *ValidationErrors) map[string]any {
	out := map[string]any{}
	for _, p := range params {
		rawVal, present := raw[p.Name]
		name := qualify(prefix, p.Name)

		if !present {
			if handleAbsent(p, name, out, errs) {
				continue
			}
			continue
		}
		coercePresent(p, rawVal, name, out, errs)
	}
	return out
}

// handleAbsent applies default / presence rules for a missing parameter. It
// returns true once it has decided the parameter's fate (always true here; the
// bool keeps the caller symmetric and readable).
func handleAbsent(p *Param, name string, out map[string]any, errs *ValidationErrors) bool {
	if p.HasDefault || p.DefaultFunc != nil {
		out[p.Name] = defaultValue(p)
		return true
	}
	if p.Required {
		errs.add(name, "is missing")
		// A missing required Hash also reports its missing inner requireds.
		if p.IsHash {
			reportMissingGroup(p.Group, name, errs)
		}
	}
	return true
}

// reportMissingGroup emits "is missing" for each required child of an absent
// required Hash (Grape reports "grp is missing, grp[inner] is missing").
func reportMissingGroup(params []*Param, prefix string, errs *ValidationErrors) {
	for _, p := range params {
		if p.Required {
			errs.add(qualify(prefix, p.Name), "is missing")
		}
	}
}

// coercePresent coerces and validates a present parameter.
func coercePresent(p *Param, rawVal any, name string, out map[string]any, errs *ValidationErrors) {
	switch {
	case p.IsHash:
		coerceHash(p, rawVal, name, out, errs)
	case p.IsArray:
		coerceArray(p, rawVal, name, out, errs)
	default:
		coerceLeaf(p, rawVal, name, out, errs)
	}
}

// coerceHash validates a nested Hash group.
func coerceHash(p *Param, rawVal any, name string, out map[string]any, errs *ValidationErrors) {
	nested, ok := rawVal.(map[string]any)
	if !ok {
		errs.add(name, "is invalid")
		return
	}
	sub := validateGroup(p.Group, nested, name, errs)
	validateExclusivityGroup(p.Group, nested, name, errs)
	out[p.Name] = sub
}

// coerceArray coerces each element of an array parameter to ElemType.
func coerceArray(p *Param, rawVal any, name string, out map[string]any, errs *ValidationErrors) {
	items, ok := rawVal.([]any)
	if !ok {
		errs.add(name, "is invalid")
		return
	}
	elems := make([]any, 0, len(items))
	valid := true
	for _, it := range items {
		s, isStr := it.(string)
		if !isStr {
			valid = false
			break
		}
		cv, ok := coerceScalar(p.ElemType, s)
		if !ok {
			valid = false
			break
		}
		elems = append(elems, cv)
	}
	if !valid {
		errs.add(name, "is invalid")
		return
	}
	out[p.Name] = elems
}

// coerceLeaf coerces and constraint-checks a scalar parameter.
func coerceLeaf(p *Param, rawVal any, name string, out map[string]any, errs *ValidationErrors) {
	raw, ok := rawVal.(string)
	if !ok {
		errs.add(name, "is invalid")
		return
	}
	if !checkAllowBlank(p, raw, name, errs) {
		return
	}
	val, ok := applyCoercion(p, raw)
	if !ok {
		errs.add(name, "is invalid")
		return
	}
	out[p.Name] = val
	if val == nil {
		// A blank optional scalar coerces to nil and skips value/length checks.
		return
	}
	checkConstraints(p, val, name, errs)
}

// applyCoercion runs the custom coerce_with lambda when present, else the
// built-in scalar coercion.
func applyCoercion(p *Param, raw string) (any, bool) {
	if p.CoerceWith != nil {
		return p.CoerceWith(raw)
	}
	return coerceScalar(p.scalarType(), raw)
}

// checkAllowBlank enforces allow_blank: false ("<param> is empty").
func checkAllowBlank(p *Param, raw string, name string, errs *ValidationErrors) bool {
	if p.AllowBlank != nil && !*p.AllowBlank && strings.TrimSpace(raw) == "" {
		errs.add(name, "is empty")
		return false
	}
	return true
}

// checkConstraints runs values / except_values / length / regexp on a coerced
// value. Grape stops at the first failing constraint for a parameter.
func checkConstraints(p *Param, val any, name string, errs *ValidationErrors) {
	if msg := checkValues(p, val); msg != "" {
		errs.add(name, msg)
		return
	}
	if msg := checkLength(p, val); msg != "" {
		errs.add(name, msg)
		return
	}
	if msg := checkRegexp(p, val); msg != "" {
		errs.add(name, msg)
		return
	}
}

// defaultValue resolves a parameter's default (callable or literal).
func defaultValue(p *Param) any {
	if p.DefaultFunc != nil {
		return p.DefaultFunc()
	}
	return p.Default
}

// qualify joins a prefix and name into Grape's "grp[inner]" nesting form.
func qualify(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s[%s]", prefix, name)
}

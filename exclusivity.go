// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

// validateExclusivity runs the top-level cross-parameter validators.
func validateExclusivity(set *ParamSet, raw Raw, prefix string, errs *ValidationErrors) {
	runExclusivity(set.MutuallyExclusive, set.ExactlyOneOf, set.AtLeastOneOf, set.AllOrNoneOf, raw, prefix, errs)
}

// validateExclusivityGroup runs a nested group's cross-parameter validators.
// A nested ParamSet is not modelled separately; group-level exclusivity is
// carried on the group's own ParamSet when the host supplies one. This helper is
// a no-op placeholder for groups declared without cross-validators, kept so the
// validate path is uniform.
func validateExclusivityGroup(params []*Param, raw Raw, prefix string, errs *ValidationErrors) {
	_ = params
	_ = raw
	_ = prefix
	_ = errs
}

// runExclusivity applies the four cross-parameter validators in Grape's order,
// using key-presence (a key present in raw, even with a blank value, counts).
func runExclusivity(mutex, exactly, atLeast, allOrNone [][]string, raw Raw, prefix string, errs *ValidationErrors) {
	for _, group := range mutex {
		if countPresent(group, raw) > 1 {
			errs.add(joinNames(qualifyAll(prefix, group)), "are mutually exclusive")
		}
	}
	for _, group := range exactly {
		n := countPresent(group, raw)
		if n == 0 {
			errs.add(joinNames(qualifyAll(prefix, group)), "are missing, exactly one parameter must be provided")
		} else if n > 1 {
			errs.add(joinNames(qualifyAll(prefix, group)), "are mutually exclusive")
		}
	}
	for _, group := range atLeast {
		if countPresent(group, raw) == 0 {
			errs.add(joinNames(qualifyAll(prefix, group)), "are missing, at least one parameter must be provided")
		}
	}
	for _, group := range allOrNone {
		n := countPresent(group, raw)
		if n != 0 && n != len(group) {
			errs.add(joinNames(qualifyAll(prefix, group)), "provide all or none of parameters")
		}
	}
}

// countPresent counts how many of the named parameters have a key in raw.
func countPresent(names []string, raw Raw) int {
	n := 0
	for _, name := range names {
		if _, ok := raw[name]; ok {
			n++
		}
	}
	return n
}

// qualifyAll prefixes each name with the group prefix for nested exclusivity
// messages.
func qualifyAll(prefix string, names []string) []string {
	if prefix == "" {
		return names
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = qualify(prefix, n)
	}
	return out
}

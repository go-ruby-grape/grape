// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

// validateExclusivity runs a ParamSet's four cross-parameter validators against
// raw, prefixing parameter names with prefix so nested groups report
// "grp[a], grp[b]". Presence is by key: a key present in raw (even with a blank
// value) counts, mirroring Grape.
func validateExclusivity(set *ParamSet, raw Raw, prefix string, errs *ValidationErrors) {
	for _, group := range set.MutuallyExclusive {
		if present := presentNames(group, raw); len(present) > 1 {
			errs.add(joinNames(qualifyAll(prefix, present)), "are mutually exclusive")
		}
	}
	for _, group := range set.ExactlyOneOf {
		present := presentNames(group, raw)
		switch {
		case len(present) == 0:
			errs.add(joinNames(qualifyAll(prefix, group)), "are missing, exactly one parameter must be provided")
		case len(present) > 1:
			// The over-supplied case reports only the present params, like
			// mutually_exclusive.
			errs.add(joinNames(qualifyAll(prefix, present)), "are mutually exclusive")
		}
	}
	for _, group := range set.AtLeastOneOf {
		if len(presentNames(group, raw)) == 0 {
			errs.add(joinNames(qualifyAll(prefix, group)), "are missing, at least one parameter must be provided")
		}
	}
	for _, group := range set.AllOrNoneOf {
		if n := len(presentNames(group, raw)); n != 0 && n != len(group) {
			errs.add(joinNames(qualifyAll(prefix, group)), "provide all or none of parameters")
		}
	}
}

// presentNames returns the subset of names whose key is present in raw, in the
// declared order.
func presentNames(names []string, raw Raw) []string {
	var out []string
	for _, name := range names {
		if _, ok := raw[name]; ok {
			out = append(out, name)
		}
	}
	return out
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

// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "strings"

// ValidationError is a single parameter failure: the fully-qualified parameter
// name (nested groups joined as "group[inner]") plus the message fragment Grape
// emits for it (without the parameter prefix).
type ValidationError struct {
	Param   string // e.g. "id" or "grp[inner]"
	Message string // e.g. "is missing", "is invalid"
}

// Error renders the failure the way Grape does: "<param> <message>".
func (e ValidationError) Error() string {
	return e.Param + " " + e.Message
}

// ValidationErrors is the aggregate raised as Grape::Exceptions::ValidationErrors.
// Its Error() string is the comma-joined per-parameter messages, matching the
// body Grape returns for a 400 (`{"error":"id is missing, name is invalid"}`).
type ValidationErrors struct {
	Errors []ValidationError
}

// Error joins the individual messages with ", " in declaration order.
func (v *ValidationErrors) Error() string {
	parts := make([]string, len(v.Errors))
	for i, e := range v.Errors {
		parts[i] = e.Error()
	}
	return strings.Join(parts, ", ")
}

// Empty reports whether any error was collected.
func (v *ValidationErrors) Empty() bool { return len(v.Errors) == 0 }

// add appends a failure for a parameter.
func (v *ValidationErrors) add(param, message string) {
	v.Errors = append(v.Errors, ValidationError{Param: param, Message: message})
}

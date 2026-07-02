// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

// Response is the shaped result of an endpoint: an HTTP status, a value to
// serialise, and the negotiated format. The host runs the endpoint body (that is
// the seam) and hands the outcome here to be shaped into a body via a Formatter.
type Response struct {
	Status int
	Value  any
	Format string
}

// ErrorBody is the value Grape serialises for an error! / a raised failure: a
// single "error" key (plus optional detail). It is emitted through the same
// Formatter as a normal response.
type ErrorBody struct {
	Message any // string, or a structured value for error!(hash, status)
	Status  int
}

// Value renders the error into the tree the formatter serialises: {"error": …}.
func (e ErrorBody) Value() any {
	m := NewOrderedMap()
	m.Set("error", e.Message)
	return m
}

// DefaultStatus returns the status Grape assigns to a method when the endpoint
// does not set one explicitly: 201 for POST, 200 otherwise. HEAD mirrors GET.
func DefaultStatus(method string) int {
	if method == "POST" {
		return 201
	}
	return 200
}

// ValidationErrorBody shapes a *ValidationErrors into the 400 body Grape returns:
// {"error":"<joined messages>"}. The status is always 400.
func ValidationErrorBody(v *ValidationErrors) ErrorBody {
	return ErrorBody{Message: v.Error(), Status: 400}
}

// NotFoundBody / MethodNotAllowedBody / NotAcceptableBody produce the plain
// bodies Grape returns for the routing failures. Grape returns a bare
// "404 Not Found" text for 404 and a JSON error for 405/406 under a JSON API.
func NotFoundBody() (int, string) { return 404, "404 Not Found" }

// MethodNotAllowedBody returns the 405 status and the value Grape serialises
// ({"error":"405 Not Allowed"}).
func MethodNotAllowedBody() (int, ErrorBody) {
	return 405, ErrorBody{Message: "405 Not Allowed", Status: 405}
}

// NotAcceptableBody returns the 406 status and error value.
func NotAcceptableBody() (int, ErrorBody) {
	return 406, ErrorBody{Message: "406 Not Acceptable", Status: 406}
}

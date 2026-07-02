// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "testing"

func TestErrorBodyValue(t *testing.T) {
	eb := ErrorBody{Message: "boom", Status: 500}
	v := eb.Value().(*OrderedMap)
	if got, _ := v.Get("error"); got != "boom" {
		t.Fatalf("error = %v", got)
	}
	// Serialises through the formatter as {"error":"boom"}.
	body, _ := Formatter{}.JSON(eb.Value())
	if body != `{"error":"boom"}` {
		t.Fatalf("body = %q", body)
	}
}

func TestDefaultStatus(t *testing.T) {
	if DefaultStatus("POST") != 201 {
		t.Fatal("POST -> 201")
	}
	if DefaultStatus("GET") != 200 {
		t.Fatal("GET -> 200")
	}
}

func TestValidationErrorBody(t *testing.T) {
	v := &ValidationErrors{}
	v.add("id", "is missing")
	v.add("name", "is invalid")
	eb := ValidationErrorBody(v)
	if eb.Status != 400 || eb.Message != "id is missing, name is invalid" {
		t.Fatalf("eb = %+v", eb)
	}
}

func TestRoutingBodies(t *testing.T) {
	if s, b := NotFoundBody(); s != 404 || b != "404 Not Found" {
		t.Fatalf("404 = %d %q", s, b)
	}
	if s, eb := MethodNotAllowedBody(); s != 405 || eb.Message != "405 Not Allowed" {
		t.Fatalf("405 = %d %+v", s, eb)
	}
	if s, eb := NotAcceptableBody(); s != 406 || eb.Message != "406 Not Acceptable" {
		t.Fatalf("406 = %d %+v", s, eb)
	}
}

func TestValidationErrorTypes(t *testing.T) {
	e := ValidationError{Param: "id", Message: "is missing"}
	if e.Error() != "id is missing" {
		t.Fatalf("error = %q", e.Error())
	}
	v := &ValidationErrors{}
	if !v.Empty() {
		t.Fatal("fresh errors should be empty")
	}
	v.add("a", "is invalid")
	if v.Empty() {
		t.Fatal("should not be empty after add")
	}
	if v.Error() != "a is invalid" {
		t.Fatalf("error = %q", v.Error())
	}
}

func TestResponseStruct(t *testing.T) {
	// Response is a plain data carrier the host fills; exercise its fields.
	r := Response{Status: 200, Value: "x", Format: "json"}
	if r.Status != 200 || r.Value != "x" || r.Format != "json" {
		t.Fatalf("r = %+v", r)
	}
}

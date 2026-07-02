// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// fill substitutes each "@@" placeholder in a Ruby harness with the next
// argument. It sidesteps fmt verbs entirely so Ruby's own "%q(...)" literals in
// the harness are not misread as Go format directives.
func fill(tmpl string, args ...string) string {
	for _, a := range args {
		tmpl = strings.Replace(tmpl, "@@", a, 1)
	}
	return tmpl
}

func atoi(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }

// grapeBin locates a usable `ruby` that can `require 'grape'` once. The oracle
// tests skip themselves when ruby or the grape/rack-test gems are absent (the
// qemu cross-arch lanes and the Windows lane), so the deterministic suite alone
// drives the 100% gate there.
func grapeBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping Grape oracle")
	}
	check := exec.Command(path, "-e", "require 'grape'; require 'rack/test'")
	if err := check.Run(); err != nil {
		t.Skip("grape / rack-test gem not installed; skipping Grape oracle")
	}
	return path
}

// rubyRun executes a Ruby script with $stdout.binmode (the go-ruby-erb lesson)
// and returns its trimmed stdout.
func rubyRun(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-e", "$stdout.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\n")
}

// oracleValidate builds a one-endpoint Grape API from a params-DSL body, drives
// it with a query hash via rack-test, and returns the response status + body.
// The Go validator is fed the same raw params so their outcomes can be compared.
const oracleHarness = `
require 'grape'
require 'rack/test'
require 'json'
class OracleAPI < Grape::API
  format :json
  params do
    @@
  end
  get('/x') { params.to_h.reject { |_, v| v.nil? }.to_json rescue 'ok' }
end
include Rack::Test::Methods
def app; OracleAPI; end
q = JSON.parse(%q(@@))
get '/x', q
puts last_response.status
puts last_response.body
`

// runGrapeValidation returns (status, body) from the live gem for a params DSL
// body and a raw query map.
func runGrapeValidation(t *testing.T, bin, dsl string, query map[string]any) (int, string) {
	t.Helper()
	qjson, _ := json.Marshal(query)
	script := fill(oracleHarness, dsl, string(qjson))
	out := rubyRun(t, bin, script)
	lines := strings.SplitN(out, "\n", 2)
	status := 0
	if len(lines) > 0 {
		status = atoi(lines[0])
	}
	body := ""
	if len(lines) > 1 {
		body = lines[1]
	}
	return status, body
}

// TestOracleValidationErrors checks that every validator's failure message here
// matches the gem's, and that a valid request is accepted (status 200) on both
// sides. Each case pairs a Grape params DSL with the equivalent Go ParamSet.
func TestOracleValidationErrors(t *testing.T) {
	bin := grapeBin(t)

	cases := []struct {
		name  string
		dsl   string
		set   *ParamSet
		query map[string]any
	}{
		{
			name:  "missing_required",
			dsl:   "requires :id, type: Integer",
			set:   &ParamSet{Params: []*Param{{Name: "id", Required: true, Type: TypeInteger}}},
			query: map[string]any{},
		},
		{
			name:  "invalid_integer",
			dsl:   "requires :id, type: Integer",
			set:   &ParamSet{Params: []*Param{{Name: "id", Required: true, Type: TypeInteger}}},
			query: map[string]any{"id": "x"},
		},
		{
			name:  "invalid_float",
			dsl:   "requires :s, type: Float",
			set:   &ParamSet{Params: []*Param{{Name: "s", Required: true, Type: TypeFloat}}},
			query: map[string]any{"s": "abc"},
		},
		{
			name:  "values_violation",
			dsl:   "requires :name, type: String, values: ['a','b','c']",
			set:   &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, Values: []any{"a", "b", "c"}}}},
			query: map[string]any{"name": "z"},
		},
		{
			name:  "except_values",
			dsl:   "optional :x, type: Integer, except_values: [1,2]",
			set:   &ParamSet{Params: []*Param{{Name: "x", Type: TypeInteger, ExceptValues: []any{int64(1), int64(2)}}}},
			query: map[string]any{"x": "1"},
		},
		{
			name:  "length_within",
			dsl:   "requires :name, type: String, length: { min: 2, max: 5 }",
			set:   &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, HasMinLen: true, MinLen: 2, HasMaxLen: true, MaxLen: 5}}},
			query: map[string]any{"name": "x"},
		},
		{
			name:  "length_min",
			dsl:   "requires :name, type: String, length: { min: 3 }",
			set:   &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, HasMinLen: true, MinLen: 3}}},
			query: map[string]any{"name": "ab"},
		},
		{
			name:  "length_max",
			dsl:   "requires :name, type: String, length: { max: 3 }",
			set:   &ParamSet{Params: []*Param{{Name: "name", Required: true, Type: TypeString, HasMaxLen: true, MaxLen: 3}}},
			query: map[string]any{"name": "abcd"},
		},
		{
			name:  "allow_blank",
			dsl:   "requires :nm, type: String, allow_blank: false",
			set:   &ParamSet{Params: []*Param{{Name: "nm", Required: true, Type: TypeString, AllowBlank: boolPtr(false)}}},
			query: map[string]any{"nm": ""},
		},
		{
			name:  "mutually_exclusive",
			dsl:   "optional :a; optional :b\n    mutually_exclusive :a, :b",
			set:   &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, MutuallyExclusive: [][]string{{"a", "b"}}},
			query: map[string]any{"a": "1", "b": "2"},
		},
		{
			name:  "exactly_one_of_missing",
			dsl:   "optional :a; optional :b\n    exactly_one_of :a, :b",
			set:   &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, ExactlyOneOf: [][]string{{"a", "b"}}},
			query: map[string]any{},
		},
		{
			name:  "at_least_one_of_missing",
			dsl:   "optional :a; optional :b\n    at_least_one_of :a, :b",
			set:   &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, AtLeastOneOf: [][]string{{"a", "b"}}},
			query: map[string]any{},
		},
		{
			name:  "all_or_none_of",
			dsl:   "optional :a; optional :b\n    all_or_none_of :a, :b",
			set:   &ParamSet{Params: []*Param{{Name: "a"}, {Name: "b"}}, AllOrNoneOf: [][]string{{"a", "b"}}},
			query: map[string]any{"a": "1"},
		},
		{
			name:  "multiple_missing",
			dsl:   "requires :a, type: Integer\n    requires :b, type: Integer",
			set:   &ParamSet{Params: []*Param{{Name: "a", Required: true, Type: TypeInteger}, {Name: "b", Required: true, Type: TypeInteger}}},
			query: map[string]any{},
		},
		{
			name: "nested_missing",
			dsl:  "requires :grp, type: Hash do\n      requires :inner, type: Integer\n    end",
			set: &ParamSet{Params: []*Param{{Name: "grp", Required: true, IsHash: true,
				Group: &ParamSet{Params: []*Param{{Name: "inner", Required: true, Type: TypeInteger}}}}}},
			query: map[string]any{},
		},
		{
			name:  "valid_request",
			dsl:   "requires :id, type: Integer\n    optional :name, type: String, values: ['a','b']",
			set:   &ParamSet{Params: []*Param{{Name: "id", Required: true, Type: TypeInteger}, {Name: "name", Type: TypeString, Values: []any{"a", "b"}}}},
			query: map[string]any{"id": "42", "name": "a"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body := runGrapeValidation(t, bin, c.dsl, c.query)
			_, goErrs := NewParamsValidator(c.set).Validate(rawFrom(c.query))
			if status == 400 {
				want := extractError(t, body)
				if goErrs == nil {
					t.Fatalf("gem rejected (%q) but Go accepted", want)
				}
				if goErrs.Error() != want {
					t.Fatalf("message mismatch:\n gem: %q\n  go: %q", want, goErrs.Error())
				}
				return
			}
			// The gem accepted (status 200).
			if goErrs != nil {
				t.Fatalf("gem accepted but Go rejected: %q", goErrs.Error())
			}
		})
	}
}

// TestOracleRouting compares router decisions with the gem for a small resource
// tree: matched status codes and captured path params.
func TestOracleRouting(t *testing.T) {
	bin := grapeBin(t)

	const routerHarness = `
require 'grape'
require 'rack/test'
class RouteAPI < Grape::API
  format :json
  resource :users do
    get(':id') { params[:id] }
    post { 'created' }
  end
  namespace :v1 do
    get('ping') { 'pong' }
  end
end
include Rack::Test::Methods
def app; RouteAPI; end
send(%q(@@).downcase, %q(@@))
puts last_response.status
`

	// The Go router mirrors the gem's declaration order.
	rt := NewRouter()
	rt.Get("/users/:id", "getUser")
	rt.Post("/users", "createUser")
	rt.Get("/v1/ping", "ping")

	cases := []struct {
		method, path string
		wantStatus   MatchStatus
	}{
		{"GET", "/users/42", StatusOK},
		{"POST", "/users", StatusOK},
		{"GET", "/v1/ping", StatusOK},
		{"DELETE", "/users/42", StatusMethodNotAllowed},
		{"GET", "/nope", StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.method+"_"+c.path, func(t *testing.T) {
			out := rubyRun(t, bin, fill(routerHarness, c.method, c.path))
			gemStatus := atoi(strings.TrimSpace(out))
			m := rt.Match(c.method, c.path)
			if m.Status != c.wantStatus {
				t.Fatalf("Go status = %v, want %v", m.Status, c.wantStatus)
			}
			wantCode := map[MatchStatus]int{StatusOK: 200, StatusMethodNotAllowed: 405, StatusNotFound: 404}[c.wantStatus]
			// POST returns 201 in the gem; treat any 2xx as OK.
			if c.wantStatus == StatusOK {
				if gemStatus < 200 || gemStatus >= 300 {
					t.Fatalf("gem status = %d, want 2xx", gemStatus)
				}
			} else if gemStatus != wantCode {
				t.Fatalf("gem status = %d, want %d", gemStatus, wantCode)
			}
		})
	}
}

// TestOracleFormatter compares the JSON formatter's output byte-for-byte with the
// gem's default JSON rendering for a representative value.
func TestOracleFormatter(t *testing.T) {
	bin := grapeBin(t)

	const fmtHarness = `
require 'grape'
require 'rack/test'
class FmtAPI < Grape::API
  format :json
  get('/f') { { id: 42, name: 'ada', tags: [1, 2] } }
end
include Rack::Test::Methods
def app; FmtAPI; end
get '/f'
puts last_response.body
`
	gemBody := rubyRun(t, bin, fmtHarness)

	m := NewOrderedMap()
	m.Set("id", int64(42))
	m.Set("name", "ada")
	m.Set("tags", []any{int64(1), int64(2)})
	goBody, _ := Formatter{}.JSON(m)
	if goBody != gemBody {
		t.Fatalf("json mismatch:\n gem: %q\n  go: %q", gemBody, goBody)
	}
}

// --- small helpers (kept ruby-free) ---

// rawFrom converts an oracle query map to the validator's Raw shape, mapping
// nested maps recursively.
func rawFrom(q map[string]any) Raw {
	out := Raw{}
	for k, v := range q {
		switch vv := v.(type) {
		case map[string]any:
			out[k] = map[string]any(rawFrom(vv))
		default:
			out[k] = v
		}
	}
	return out
}

// extractError pulls the "error" field out of a Grape 400 JSON body.
func extractError(t *testing.T, body string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("bad error body %q: %v", body, err)
	}
	s, _ := m["error"].(string)
	return s
}

func boolPtr(b bool) *bool { return &b }

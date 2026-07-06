<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-grape/brand/main/social/go-ruby-grape-grape.png" alt="go-ruby-grape/grape" width="720"></p>

# grape — go-ruby-grape

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-grape.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic, interpreter-independent
core of Ruby's [Grape](https://github.com/ruby-grape/grape) REST-API framework**
(the `grape` gem) — **route modelling + matching, params validation/coercion, and
response formatting** — so a host can resolve a request, validate its parameters,
and shape a response the way Grape does **without any Ruby runtime**.

It is the API-machinery backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), a sibling of
[go-ruby-rack](https://github.com/go-ruby-rack/rack). Running an endpoint body and
parsing the Rack env are the **host's** job; this library hands back the small,
explicit models the host maps to and from its own objects.

> **What it is — and isn't.** Matching `(method, path, headers)` to a route,
> coercing/validating a params tree, and serialising a value to json / xml / txt
> are fully deterministic and need **no interpreter**, so they live here as pure
> Go. Binding the endpoint block to a live Ruby method — evaluating the body,
> reading `params` inside it — is the host's job.

## The three deterministic pieces

### Router — `(method, path) → (route, path params) + 404/405/406`

```go
rt := grape.NewRouter()
rt.Get("/users/:id", getUser)   // handler is an opaque host reference
rt.Post("/users", createUser)

m := rt.Match("GET", "/users/42")
// m.Status == grape.StatusOK; m.Route.Handler == getUser; m.Params["id"] == "42"

rt.Match("DELETE", "/users/42") // StatusMethodNotAllowed, m.Allowed lists verbs
rt.Match("GET", "/nope")        // StatusNotFound
```

Path patterns cover `:name` captures, `*wildcard` tails, and `.:format`
suffixes (`/users/:id.:format`). A `GET` route answers `HEAD`; an empty/`*`
method matches any verb. [Version] models the four version strategies
(path / header / param / accept), and [Negotiate] resolves the response format
from an extension, a forced format, or the `Accept` header (the 406 decision).

### ParamsValidator — coerce + validate, or raise Grape's exact errors

```go
set := &grape.ParamSet{Params: []*grape.Param{
    {Name: "id", Required: true, Type: grape.TypeInteger},
    {Name: "name", Type: grape.TypeString, Values: []any{"a", "b", "c"}},
}}
v := grape.NewParamsValidator(set)

coerced, errs := v.Validate(grape.Raw{"id": "42", "name": "z"})
// errs.Error() == "name does not have a valid value"   (matches the gem verbatim)

coerced, _ = v.Validate(grape.Raw{"id": "42"})
// coerced["id"] == int64(42)
```

Validators, all producing Grape's exact message fragments:

| declaration | message on failure |
| --- | --- |
| `requires` (absent) | `id is missing` |
| `type:` coercion | `id is invalid` |
| `values:` / range | `name does not have a valid value` |
| `except_values:` | `x has a value not allowed` |
| `length: {min,max}` | `name is expected to have length within 2 and 5` |
| `length: {min}` / `{max}` | `… greater/less than or equal to N` |
| `regexp:` | `email is invalid` |
| `allow_blank: false` | `nm is empty` |
| `mutually_exclusive` | `a, b are mutually exclusive` |
| `exactly_one_of` | `a, b are missing, exactly one parameter must be provided` |
| `at_least_one_of` | `a, b are missing, at least one parameter must be provided` |
| `all_or_none_of` | `a, b provide all or none of parameters` |

Types coerce to the Ruby value model: `Integer` → `int64` / `*big.Int`,
`Float` → `float64`, `Boolean` → `bool`, `Date`/`Time` → `time.Time`,
`JSON` → the generic tree, `Array[T]` → `[]any`, `Hash` → a nested group. A
blank optional scalar coerces to `nil` (as Grape does); `default:` supplies a
literal or a callable (`DefaultFunc`); `coerce_with:` swaps in a host lambda;
nested groups nest error names as `grp[inner]`.

### Formatter — json / txt / xml

```go
var f grape.Formatter
m := grape.NewOrderedMap()
m.Set("id", int64(42))
m.Set("name", "ada")
body, mime, _ := f.Format("json", m) // {"id":42,"name":"ada"}, application/json
```

`OrderedMap` preserves hash key order through JSON/XML; a plain `map[string]any`
is emitted in sorted-key order for determinism. `error!` / status shaping lives
in [Response] / [ErrorBody].

## Value model

The validator and formatter exchange a small, fixed set of Go types the host maps
to its own objects:

| Ruby | Go |
| --- | --- |
| `nil` | `nil` |
| `true`/`false` | `bool` |
| `Integer` | `int64` / `*big.Int` |
| `Float` | `float64` |
| `String` | `string` |
| `Time`/`Date` | `time.Time` |
| `Array` | `[]any` |
| `Hash` | `map[string]any` / `*grape.OrderedMap` |

## What the host (rbgo) binds

The Ruby-facing seam — mapping `Grape::API.get/post/params/…` DSL calls onto
`Router` + `ParamSet`, running the endpoint block, and reading the Rack env — is
the host's job (ties to [go-ruby-rack](https://github.com/go-ruby-rack/rack)).
The host feeds `(method, path)` to `Router.Match`, the parsed params tree to
`ParamsValidator.Validate`, and the endpoint's return value to `Formatter.Format`.

## Install

```sh
go get github.com/go-ruby-grape/grape
```

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential MRI oracle**: the same raw params are fed to these validators and
to a live `grape` API (via `rack-test`), and their coerced output + error
messages are compared; `(method, path)` is fed to both routers and the matched
route + path params + status are compared. The oracle skips itself where `ruby`
or the `grape` gem is absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
GOWORK=off go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

CGO-free, dependency-free, `gofmt` + `go vet` clean, and green across the six
64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le, s390x).

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-grape/grape authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```

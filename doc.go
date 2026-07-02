// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package grape is a pure-Go (no cgo) reimplementation of the deterministic,
// interpreter-independent core of Ruby's Grape REST-API framework (the `grape`
// gem): route modelling + matching, params validation/coercion, and response
// formatting.
//
// It is the API-machinery backend for go-embedded-ruby, but is a standalone,
// reusable module with no dependency on the Ruby runtime — a sibling of
// go-ruby-rack. Three deterministic pieces live here:
//
//   - A [Router]: add routes (get/post/…, path patterns with :params, .:format
//     suffixes and *wildcards, namespaces, versioning) and resolve
//     (method, path, headers) → (matched route, path params) plus the 404 / 405
//     / 406 decisions.
//   - A [ParamsValidator]: turn a params declaration (requires/optional, type
//     coercion, values, length, regexp, mutual-exclusion, defaults, nested
//     groups) plus raw params into the coerced params hash or a
//     [ValidationErrors] carrying Grape's exact messages.
//   - A [Formatter]: json / txt / xml serialisation of the response value shape.
//
// Binding endpoint blocks to live Ruby objects and parsing the Rack env are the
// host's job (rbgo, tied to go-ruby-rack); this library hands back a small,
// explicit value model the host maps to and from its own objects.
package grape

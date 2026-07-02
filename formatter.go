// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Formatter serialises a response value to the wire form for a format symbol.
// The value model is the generic Go tree the host produces from the endpoint's
// return value: nil, bool, integers, floats, string, []any, and map[string]any
// (or *OrderedMap for order-preserving hashes). Binding a live Ruby object to
// this tree is the host's job; the formatter only serialises the tree.
type Formatter struct{}

// OrderedMap is an insertion-ordered string-keyed map, so a hash's key order
// survives JSON/XML serialisation (a plain map[string]any is emitted in sorted
// key order for determinism).
type OrderedMap struct {
	keys []string
	vals map[string]any
}

// NewOrderedMap returns an empty ordered map.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{vals: map[string]any{}}
}

// Set inserts or updates a key, preserving first-insertion order.
func (m *OrderedMap) Set(key string, val any) {
	if _, ok := m.vals[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.vals[key] = val
}

// Get returns the value for key.
func (m *OrderedMap) Get(key string) (any, bool) {
	v, ok := m.vals[key]
	return v, ok
}

// Keys returns the keys in insertion order.
func (m *OrderedMap) Keys() []string { return m.keys }

// Len returns the number of entries.
func (m *OrderedMap) Len() int { return len(m.keys) }

// Format serialises v for the given format symbol ("json", "txt", "xml"). It
// returns the body and the Content-Type MIME. An unknown format is an error.
func (f Formatter) Format(format string, v any) (body string, mime string, err error) {
	switch format {
	case "json", "jsonapi", "serializable_hash":
		b, err := f.JSON(v)
		return b, MimeFor(format), err
	case "txt":
		return f.Txt(v), MimeFor("txt"), nil
	case "xml":
		b, err := f.XML(v)
		return b, MimeFor("xml"), err
	}
	return "", "", fmt.Errorf("grape: unknown format %q", format)
}

// JSON renders v as compact JSON, matching Grape's default (MultiJson) output:
// no spaces, sorted keys for plain maps, insertion order for OrderedMap.
func (f Formatter) JSON(v any) (string, error) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// writeJSON emits a value in Grape's compact style, handling OrderedMap key
// order explicitly (encoding/json cannot).
func writeJSON(buf *bytes.Buffer, v any) error {
	switch val := v.(type) {
	case *OrderedMap:
		return writeJSONOrdered(buf, val)
	case map[string]any:
		return writeJSONMap(buf, val)
	case []any:
		return writeJSONArray(buf, val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}
}

func writeJSONOrdered(buf *bytes.Buffer, m *OrderedMap) error {
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeJSONKey(buf, k)
		if err := writeJSON(buf, m.vals[k]); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

func writeJSONMap(buf *bytes.Buffer, m map[string]any) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeJSONKey(buf, k)
		if err := writeJSON(buf, m[k]); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

func writeJSONArray(buf *bytes.Buffer, a []any) error {
	buf.WriteByte('[')
	for i, e := range a {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeJSON(buf, e); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

func writeJSONKey(buf *bytes.Buffer, k string) {
	kb, _ := json.Marshal(k)
	buf.Write(kb)
	buf.WriteByte(':')
}

// Txt renders v the way Grape's Txt formatter does: a String is emitted verbatim,
// anything else via its to_s form. For the generic tree that is a JSON-free
// inspection; the host may supply richer to_s via the value already being a
// string. Here a string passes through and other values use a Go string form.
func (f Formatter) Txt(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// XML renders v in the ActiveSupport-compatible shape Grape's Xml formatter
// emits: a map becomes a "<hash>" element whose children are its typed keys, an
// array nests repeated child elements, and scalars carry a type attribute.
func (f Formatter) XML(v any) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	writeXMLRoot(&buf, v)
	return buf.String(), nil
}

func writeXMLRoot(buf *bytes.Buffer, v any) {
	switch val := v.(type) {
	case *OrderedMap:
		buf.WriteString("<hash>\n")
		for _, k := range val.keys {
			writeXMLElem(buf, k, val.vals[k], 1)
		}
		buf.WriteString("</hash>\n")
	case map[string]any:
		buf.WriteString("<hash>\n")
		keys := sortedKeys(val)
		for _, k := range keys {
			writeXMLElem(buf, k, val[k], 1)
		}
		buf.WriteString("</hash>\n")
	default:
		writeXMLElem(buf, "hash", v, 0)
	}
}

// writeXMLElem emits one <name>…</name> element at the given indent depth.
func writeXMLElem(buf *bytes.Buffer, name string, v any, depth int) {
	pad := strings.Repeat("  ", depth)
	switch val := v.(type) {
	case *OrderedMap:
		fmt.Fprintf(buf, "%s<%s type=\"hash\">\n", pad, name)
		for _, k := range val.keys {
			writeXMLElem(buf, k, val.vals[k], depth+1)
		}
		fmt.Fprintf(buf, "%s</%s>\n", pad, name)
	case map[string]any:
		fmt.Fprintf(buf, "%s<%s type=\"hash\">\n", pad, name)
		for _, k := range sortedKeys(val) {
			writeXMLElem(buf, k, val[k], depth+1)
		}
		fmt.Fprintf(buf, "%s</%s>\n", pad, name)
	case []any:
		fmt.Fprintf(buf, "%s<%s type=\"array\">\n", pad, name)
		for _, e := range val {
			writeXMLElem(buf, name, e, depth+1)
		}
		fmt.Fprintf(buf, "%s</%s>\n", pad, name)
	default:
		typ := xmlType(v)
		if v == nil {
			fmt.Fprintf(buf, "%s<%s nil=\"true\"/>\n", pad, name)
			return
		}
		fmt.Fprintf(buf, "%s<%s type=\"%s\">%s</%s>\n", pad, name, typ, xmlScalar(v), name)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// xmlType maps a scalar to ActiveSupport's type attribute.
func xmlType(v any) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case int, int64:
		return "integer"
	case float64, float32:
		return "float"
	}
	return "string"
}

// xmlScalar renders a scalar's text body.
func xmlScalar(v any) string {
	switch val := v.(type) {
	case string:
		return xmlEscape(val)
	default:
		return fmt.Sprint(val)
	}
}

// xmlEscape escapes the five XML predefined entities.
func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

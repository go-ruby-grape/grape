// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import "testing"

func TestOrderedMap(t *testing.T) {
	m := NewOrderedMap()
	m.Set("b", 2)
	m.Set("a", 1)
	m.Set("b", 3) // update keeps position, does not re-append
	if got := m.Keys(); len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Fatalf("keys = %v", got)
	}
	if m.Len() != 2 {
		t.Fatalf("len = %d", m.Len())
	}
	if v, ok := m.Get("b"); !ok || v != 3 {
		t.Fatalf("b = %v %v", v, ok)
	}
	if _, ok := m.Get("missing"); ok {
		t.Fatalf("missing should be absent")
	}
}

func TestFormatJSON(t *testing.T) {
	var f Formatter
	m := NewOrderedMap()
	m.Set("id", int64(42))
	m.Set("name", "ada")
	m.Set("arr", []any{int64(1), int64(2)})
	inner := NewOrderedMap()
	inner.Set("k", "v")
	m.Set("nested", inner)
	body, mime, err := f.Format("json", m)
	if err != nil {
		t.Fatal(err)
	}
	if mime != "application/json" {
		t.Fatalf("mime = %q", mime)
	}
	want := `{"id":42,"name":"ada","arr":[1,2],"nested":{"k":"v"}}`
	if body != want {
		t.Fatalf("json = %q want %q", body, want)
	}
	// Plain map is emitted in sorted-key order.
	body, _, _ = f.Format("json", map[string]any{"b": 2, "a": 1})
	if body != `{"a":1,"b":2}` {
		t.Fatalf("sorted map = %q", body)
	}
	// Scalars and arrays at the top level.
	body, _ = f.JSON([]any{"x", map[string]any{"z": 1}})
	if body != `["x",{"z":1}]` {
		t.Fatalf("array = %q", body)
	}
	body, _ = f.JSON(nil)
	if body != "null" {
		t.Fatalf("nil = %q", body)
	}
}

func TestFormatJSONError(t *testing.T) {
	var f Formatter
	// A channel is not JSON-serialisable: writeJSON's default branch errors.
	if _, err := f.JSON(make(chan int)); err == nil {
		t.Fatal("expected error for unserialisable value")
	}
	if _, err := f.JSON([]any{make(chan int)}); err == nil {
		t.Fatal("expected error in array element")
	}
	if _, err := f.JSON(map[string]any{"x": make(chan int)}); err == nil {
		t.Fatal("expected error in map value")
	}
	om := NewOrderedMap()
	om.Set("x", make(chan int))
	if _, err := f.JSON(om); err == nil {
		t.Fatal("expected error in ordered-map value")
	}
	// Format() surfaces the JSON error too.
	if _, _, err := f.Format("json", make(chan int)); err == nil {
		t.Fatal("expected Format error")
	}
}

func TestFormatTxt(t *testing.T) {
	var f Formatter
	body, mime, err := f.Format("txt", "hello")
	if err != nil || mime != "text/plain" || body != "hello" {
		t.Fatalf("txt = %q %q %v", body, mime, err)
	}
	// Non-string uses fmt.Sprint.
	if got := f.Txt(int64(7)); got != "7" {
		t.Fatalf("txt int = %q", got)
	}
}

func TestFormatXML(t *testing.T) {
	var f Formatter
	m := NewOrderedMap()
	m.Set("a", int64(1))
	m.Set("b", []any{int64(1), int64(2)})
	body, mime, err := f.Format("xml", m)
	if err != nil || mime != "application/xml" {
		t.Fatalf("mime=%q err=%v", mime, err)
	}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<hash>
  <a type="integer">1</a>
  <b type="array">
    <b type="integer">1</b>
    <b type="integer">2</b>
  </b>
</hash>
`
	if body != want {
		t.Fatalf("xml =\n%s\nwant\n%s", body, want)
	}
}

func TestFormatXMLShapes(t *testing.T) {
	var f Formatter
	// Plain map root (sorted keys), nested hash, bool/float/string/nil scalars.
	body, _ := f.XML(map[string]any{
		"z": true,
		"a": 3.5,
		"s": "x<y",
		"n": nil,
		"h": map[string]any{"inner": "v"},
	})
	want := `<?xml version="1.0" encoding="UTF-8"?>
<hash>
  <a type="float">3.5</a>
  <h type="hash">
    <inner type="string">v</inner>
  </h>
  <n nil="true"/>
  <s type="string">x&lt;y</s>
  <z type="boolean">true</z>
</hash>
`
	if body != want {
		t.Fatalf("xml =\n%s\nwant\n%s", body, want)
	}
	// Ordered nested map inside an ordered root.
	root := NewOrderedMap()
	inner := NewOrderedMap()
	inner.Set("k", "v")
	root.Set("h", inner)
	body, _ = f.XML(root)
	wantOrd := `<?xml version="1.0" encoding="UTF-8"?>
<hash>
  <h type="hash">
    <k type="string">v</k>
  </h>
</hash>
`
	if body != wantOrd {
		t.Fatalf("ordered xml =\n%s", body)
	}
	// A scalar root falls through to a bare <hash> element.
	body, _ = f.XML(int64(5))
	wantScalar := `<?xml version="1.0" encoding="UTF-8"?>
<hash type="integer">5</hash>
`
	if body != wantScalar {
		t.Fatalf("scalar root =\n%s", body)
	}
}

func TestFormatUnknown(t *testing.T) {
	var f Formatter
	if _, _, err := f.Format("yaml", "x"); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

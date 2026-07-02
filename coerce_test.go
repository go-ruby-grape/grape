// Copyright (c) the go-ruby-grape/grape authors
//
// SPDX-License-Identifier: BSD-3-Clause

package grape

import (
	"math/big"
	"testing"
)

func TestCoerceBoolean(t *testing.T) {
	truthy := []string{"true", "1", "t", "yes", "y", "on", "TRUE", "On"}
	for _, s := range truthy {
		if v, ok := coerceBoolean(s); !ok || v != true {
			t.Fatalf("boolean(%q) = %v %v, want true", s, v, ok)
		}
	}
	falsey := []string{"false", "0", "f", "no", "n", "off", "FALSE", "Off"}
	for _, s := range falsey {
		if v, ok := coerceBoolean(s); !ok || v != false {
			t.Fatalf("boolean(%q) = %v %v, want false", s, v, ok)
		}
	}
	if _, ok := coerceBoolean("maybe"); ok {
		t.Fatal("maybe should be invalid")
	}
}

func TestCoerceScalarUnknownType(t *testing.T) {
	if _, ok := coerceScalar(Type("Bogus"), "x"); ok {
		t.Fatal("unknown type should be invalid")
	}
	// Empty string coerces to nil for a non-string type even when unknown falls
	// through the blank guard only for known types; a bare String keeps "".
	if v, ok := coerceScalar(TypeString, ""); !ok || v != "" {
		t.Fatalf("empty string = %v %v", v, ok)
	}
	if v, ok := coerceScalar("", "raw"); !ok || v != "raw" {
		t.Fatalf("untyped passthrough = %v %v", v, ok)
	}
}

func TestCoerceIntegerBig(t *testing.T) {
	big1, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	v, ok := coerceInteger("123456789012345678901234567890")
	if !ok {
		t.Fatal("big int should coerce")
	}
	if got := v.(*big.Int); got.Cmp(big1) != 0 {
		t.Fatalf("big = %v", got)
	}
	if _, ok := coerceInteger("12.5"); ok {
		t.Fatal("decimal should not be an integer")
	}
}

func TestLengthOfArray(t *testing.T) {
	set := &ParamSet{Params: []*Param{{Name: "tags", Required: true, IsArray: true, ElemType: TypeString, HasMaxLen: true, MaxLen: 2}}}
	if _, e := errString(t, set, Raw{"tags": []any{"a", "b", "c"}}); e != "tags is expected to have length less than or equal to 2" {
		t.Fatalf("got %q", e)
	}
	if _, e := errString(t, set, Raw{"tags": []any{"a"}}); e != "" {
		t.Fatalf("got %q", e)
	}
}

func TestValueComparisonHelpers(t *testing.T) {
	// int vs int64 vs *big.Int all compare equal through valuesEqual.
	if !valuesEqual(int(5), int64(5)) {
		t.Fatal("int == int64")
	}
	if !valuesEqual(big.NewInt(5), int(5)) {
		t.Fatal("big == int")
	}
	// A big-int on one side and a non-int on the other is unequal.
	if valuesEqual(big.NewInt(5), "5") {
		t.Fatal("big != string")
	}
	// Non-int values fall back to == .
	if !valuesEqual("x", "x") {
		t.Fatal("string == string")
	}
	// toInt64 across the three integer shapes + overflow.
	if n, ok := toInt64(int(3)); !ok || n != 3 {
		t.Fatal("toInt64 int")
	}
	if n, ok := toInt64(big.NewInt(9)); !ok || n != 9 {
		t.Fatal("toInt64 big")
	}
	huge, _ := new(big.Int).SetString("99999999999999999999999999", 10)
	if _, ok := toInt64(huge); ok {
		t.Fatal("overflow big should not fit int64")
	}
	if _, ok := toInt64("x"); ok {
		t.Fatal("non-int toInt64")
	}
	// toBigInt across int/int64/big and the miss case.
	if b, ok := toBigInt(int(1)); !ok || b.Int64() != 1 {
		t.Fatal("toBigInt int")
	}
	if _, ok := toBigInt("x"); ok {
		t.Fatal("non-int toBigInt")
	}
}

func TestExceptValuesRange(t *testing.T) {
	// Range check against an int coerced from a big-int-typed value goes through
	// toInt64's *big.Int arm.
	set := &ParamSet{Params: []*Param{{Name: "n", Type: TypeInteger, HasRange: true, RangeMin: 1, RangeMax: 10}}}
	if _, e := errString(t, set, Raw{"n": "3"}); e != "" {
		t.Fatalf("got %q", e)
	}
}

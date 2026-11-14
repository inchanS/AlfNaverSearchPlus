// Package jsonx provides small helpers for navigating dynamically-typed JSON
// decoded into interface{} values. Naver's autocomplete APIs return deeply
// nested, heterogeneous arrays, so typed structs are impractical; these helpers
// mirror Python's forgiving indexing (returning zero values on any mismatch).
package jsonx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Decode parses JSON bytes into a generic value, keeping numbers as
// json.Number so coordinates and IDs preserve their exact textual form.
func Decode(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("expected a single JSON value")
	}
	return v, nil
}

// DecodeObject rejects null, arrays and scalars where an API object is expected.
func DecodeObject(data []byte) (map[string]any, error) {
	v, err := Decode(data)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a JSON object")
	}
	return m, nil
}

// Arr returns v as a slice, or nil if v is not an array.
func Arr(v any) []any {
	a, _ := v.([]any)
	return a
}

// At returns the i-th element of v (if v is an array and i is in range), else nil.
func At(v any, i int) any {
	a := Arr(v)
	if i >= 0 && i < len(a) {
		return a[i]
	}
	return nil
}

// Obj returns v as a map, or nil if v is not an object.
func Obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// Get returns v[k] when v is an object, else nil.
func Get(v any, k string) any {
	return Obj(v)[k]
}

// Str returns v as a string, or "" if v is not a string.
func Str(v any) string {
	s, _ := v.(string)
	return s
}

// AsStr coerces strings, numbers and bools to their textual form. Numbers keep
// their source representation (json.Number), matching Python's str() output for
// the values used here.
func AsStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	}
	return ""
}

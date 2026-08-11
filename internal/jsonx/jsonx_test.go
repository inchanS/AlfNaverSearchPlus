package jsonx

import "testing"

func TestNavigation(t *testing.T) {
	// Mirrors the shape search/terms return: {"items": [ [ ["a"], ["b"] ] ]}
	root, err := Decode([]byte(`{"items":[[["a",1],["b",2]]]}`))
	if err != nil {
		t.Fatal(err)
	}
	first := At(Get(root, "items"), 0)
	if got := Str(At(At(first, 0), 0)); got != "a" {
		t.Fatalf("nested index = %q, want a", got)
	}
	// Out-of-range and type mismatches return zero values, never panic.
	if At(first, 99) != nil {
		t.Fatal("out-of-range index should be nil")
	}
	if Str(At(first, 0)) != "" {
		t.Fatal("Str of an array should be empty")
	}
}

func TestAsStrPreservesNumbers(t *testing.T) {
	root, _ := Decode([]byte(`{"lat":"37.5665851","x":126.978,"id":123,"ok":true}`))
	if got := AsStr(Get(root, "lat")); got != "37.5665851" {
		t.Fatalf("lat = %q", got)
	}
	// json.Number keeps the exact source text (no float rounding).
	if got := AsStr(Get(root, "x")); got != "126.978" {
		t.Fatalf("x = %q", got)
	}
	if got := AsStr(Get(root, "id")); got != "123" {
		t.Fatalf("id = %q", got)
	}
	if got := AsStr(Get(root, "ok")); got != "true" {
		t.Fatalf("ok = %q", got)
	}
}

package cache

import (
	"os"
	"testing"
	"time"
)

func TestKey(t *testing.T) {
	// Deterministic md5-based key with a stable prefix.
	if got := Key("nav", "hello"); got != "nav_5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("Key = %q", got)
	}
	// A word containing '/' still yields a filesystem-safe key.
	if got := Key("navmap", "a/b"); len(got) != len("navmap_")+32 {
		t.Fatalf("unexpected key length for slash word: %q", got)
	}
}

func TestReadWriteFreshness(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())

	if err := Write("k", []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}

	// Within maxAge -> hit.
	if data, ok := Read("k", time.Hour); !ok || string(data) != `{"v":1}` {
		t.Fatalf("fresh read failed: %q ok=%v", data, ok)
	}

	// maxAge 0 -> never expires, always a hit when present.
	if _, ok := Read("k", 0); !ok {
		t.Fatal("maxAge=0 should always hit when file exists")
	}

	// Stale (maxAge shorter than age) -> miss.
	old := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(path("k"), old, old)
	if _, ok := Read("k", time.Hour); ok {
		t.Fatal("stale entry should miss")
	}

	// Missing key -> miss.
	if _, ok := Read("nope", time.Hour); ok {
		t.Fatal("missing key should miss")
	}
}

func TestCleanup(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())

	// Seed four entries: a stale query file, a fresh query file, a stale internal
	// bookkeeping file ("__" prefix), and a stale non-JSON file.
	for _, k := range []string{"nav_stale", "nav_fresh", "__update_info"} {
		if err := Write(k, []byte("{}")); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path("nav_stale")[:len(path("nav_stale"))-len(".json")]+".txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-48 * time.Hour)
	for _, k := range []string{"nav_stale", "__update_info"} {
		if err := os.Chtimes(path(k), old, old); err != nil {
			t.Fatal(err)
		}
	}

	Cleanup(24 * time.Hour)

	// Stale query file pruned.
	if _, ok := Read("nav_stale", 0); ok {
		t.Error("stale query file should be removed")
	}
	// Fresh query file kept.
	if _, ok := Read("nav_fresh", 0); !ok {
		t.Error("fresh query file should be kept")
	}
	// Internal "__" file preserved even when stale.
	if _, ok := Read("__update_info", 0); !ok {
		t.Error("__-prefixed bookkeeping file should be preserved")
	}
	// Non-.json file untouched.
	txt := path("nav_stale")[:len(path("nav_stale"))-len(".json")] + ".txt"
	if _, err := os.Stat(txt); err != nil {
		t.Error("non-.json file should be untouched")
	}
}

func TestCachedUsesLoaderOnce(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())

	calls := 0
	loader := func() ([]byte, error) {
		calls++
		return []byte("payload"), nil
	}

	for i := 0; i < 3; i++ {
		data, err := Cached("k", time.Hour, loader)
		if err != nil || string(data) != "payload" {
			t.Fatalf("Cached returned %q err=%v", data, err)
		}
	}
	if calls != 1 {
		t.Fatalf("loader called %d times, want 1", calls)
	}

	// Nil loader on a miss returns no data without error.
	if data, err := Cached("missing", time.Hour, nil); err != nil || data != nil {
		t.Fatalf("nil loader miss = %q err=%v", data, err)
	}
}

// Package cache is a small on-disk cache replacing wf.cached_data / cache_data.
// Entries are JSON files under Alfred's per-workflow cache directory, with
// freshness judged by file mtime.
package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dir returns Alfred's workflow cache directory, or a temp fallback when the
// workflow env var is absent (e.g. local runs and tests).
func dir() string {
	if d := os.Getenv("alfred_workflow_cache"); d != "" {
		return d
	}
	return filepath.Join(os.TempDir(), "naversearchplus-cache")
}

func path(key string) string {
	return filepath.Join(dir(), key+".json")
}

// Key builds a filesystem-safe cache key: prefix + md5(word). The word may
// contain '/' or other path separators, so it is always hashed.
func Key(prefix, word string) string {
	sum := md5.Sum([]byte(word))
	return prefix + "_" + hex.EncodeToString(sum[:])
}

// Read returns the cached bytes for key when present and still fresh. A maxAge
// of 0 means "never expires": any existing entry is returned.
func Read(key string, maxAge time.Duration) ([]byte, bool) {
	p := path(key)
	f, err := os.Open(p)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, false
	}
	if maxAge > 0 && time.Since(fi.ModTime()) > maxAge {
		return nil, false
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Write atomically replaces an entry. Readers see either complete version,
// including when another Script Filter process is stopped during a write.
func Write(key string, data []byte) error {
	d := dir()
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(d, ".cache-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(d, key+".json"))
}

// Cleanup deletes stale query-cache files whose mtime is older than maxAge,
// bounding the cache directory's growth (each distinct search leaves a small
// JSON file that is otherwise never removed). Internal bookkeeping entries whose
// names start with "__" (e.g. the update info/stamp) are always preserved, so a
// stale-but-current update notice survives pruning. Errors are ignored: cleanup
// is best-effort housekeeping run from the background worker, never on the
// foreground autocomplete path.
func Cleanup(maxAge time.Duration) {
	entries, err := os.ReadDir(dir())
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, "__") ||
			(!strings.HasSuffix(name, ".json") && !strings.HasPrefix(name, ".cache-")) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir(), name))
		}
	}
}

// Load validates cached and fetched data using decode. Invalid entries are
// discarded and retried; invalid responses are never cached. A cache write
// failure is diagnostic only and must not discard a successful response.
func Load[T any](key string, maxAge time.Duration, loader func() ([]byte, error), decode func([]byte) (T, error)) (T, error) {
	if data, ok := Read(key, maxAge); ok {
		if value, err := decode(data); err == nil {
			return value, nil
		}
		_ = os.Remove(path(key))
	}
	var zero T
	if loader == nil {
		return zero, nil
	}
	data, err := loader()
	if err != nil {
		return zero, err
	}
	value, err := decode(data)
	if err != nil {
		return zero, err
	}
	if err := Write(key, data); err != nil {
		fmt.Fprintf(os.Stderr, "cache: could not save %s: %v\n", key, err)
	}
	return value, nil
}

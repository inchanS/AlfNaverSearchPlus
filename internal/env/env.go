// Package env reads Alfred workflow configuration from environment variables,
// applying the same defaults the previous Python scripts used.
package env

import (
	"os"
	"strconv"
	"time"
)

// Default coordinates fall back to Seoul City Hall, matching naver_map_common.
const (
	DefaultLatitude  = "37.5665851"
	DefaultLongitude = "126.9782038"
)

// Str returns the env var or def when unset/empty.
func Str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// CacheAge returns the `cache_age` env var (seconds) as a Duration, defaulting
// to 30s when unset or invalid.
func CacheAge() time.Duration {
	n, err := strconv.Atoi(os.Getenv("cache_age"))
	if err != nil || n <= 0 {
		n = 30
	}
	return time.Duration(n) * time.Second
}

// Latitude / Longitude return the configured default coordinates.
func Latitude() string  { return Str("latitude", DefaultLatitude) }
func Longitude() string { return Str("longitude", DefaultLongitude) }

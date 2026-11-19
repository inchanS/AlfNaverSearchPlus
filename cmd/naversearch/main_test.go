package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestToggleDispatchBeforeSearch(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("alfred_workflow_data", dataDir)
	cacheDir := filepath.Join(t.TempDir(), "unused-cache")
	t.Setenv("alfred_workflow_cache", cacheDir)
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	for _, tc := range []struct {
		query string
		want  bool
	}{
		{"workflow:noautoupdate", false},
		{"workflow:autoupdate", true},
	} {
		os.Args = []string{"naversearch", "search", tc.query}
		main()
		data, err := os.ReadFile(filepath.Join(dataDir, "update-settings.json"))
		var settings struct {
			Enabled bool `json:"automatic_updates"`
		}
		if err != nil || json.Unmarshal(data, &settings) != nil || settings.Enabled != tc.want {
			t.Fatalf("%s did not reach toggle handler: %s, %v", tc.query, data, err)
		}
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("toggle ran search or spawned a worker: %v", err)
	}
}

func TestNormalizeArgs(t *testing.T) {
	// Build NFD (decomposed) inputs explicitly via code points, independent of
	// this file's own encoding.
	//   "스타" as conjoining jamo: ᄉ ᅳ ᄐ ᅡ
	starNFD := string([]rune{0x1109, 0x1173, 0x1110, 0x1161})
	//   "がっこう": か + combining dakuten, then っこう
	gakkoNFD := string([]rune{0x304B, 0x3099, 0x3063, 0x3053, 0x3046})

	args := []string{"map", "doNotUseIP", starNFD, gakkoNFD, "ascii"}
	normalizeArgs(args)

	if want := "스타"; args[2] != want {
		t.Errorf("Hangul not composed to NFC: got % x, want % x", []byte(args[2]), []byte(want))
	}
	if want := "がっこう"; args[3] != want {
		t.Errorf("Japanese dakuten not composed to NFC: got % x, want % x", []byte(args[3]), []byte(want))
	}
	// Handler names and ASCII are untouched.
	if args[0] != "map" || args[1] != "doNotUseIP" || args[4] != "ascii" {
		t.Errorf("ASCII args changed: %v", args)
	}
}

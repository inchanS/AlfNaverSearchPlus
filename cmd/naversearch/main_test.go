package main

import (
	"encoding/json"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"io"
	"os"
	"strings"
	"testing"
)

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

func TestMapUILaunchTreatsUpdateMagicAsLiteralQuery(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	t.Setenv("alfred_workflow_version", "3.0.0")
	if err := cache.Write("__worker_run", []byte("test")); err != nil {
		t.Fatal(err)
	}
	if err := cache.Write("__update_info", []byte(`{"version":"3.1.0","available":true}`)); err != nil {
		t.Fatal(err)
	}
	oldArgs, oldOut := os.Args, os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer func() { os.Args = oldArgs; os.Stdout = oldOut }()
	os.Args = []string{"naversearch", "map-ui-launch", "useIP", "workflow:update"}
	os.Stdout = writer
	main()
	writer.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Workflow struct {
			Args []string `json:"arg"`
		} `json:"alfredworkflow"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Workflow.Args) != 3 || out.Workflow.Args[0] != "workflow:update" || out.Workflow.Args[1] != "useIP" || !strings.Contains(out.Workflow.Args[2], "새 버전 있음") {
		t.Fatalf("wrong UI launch context: %s", data)
	}
}

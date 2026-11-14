package workflow_test

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Only XML structure is needed to read the checked-in workflow configuration;
// keep these integration tests independent of a platform-specific plist tool.
type plistNode struct {
	XMLName  xml.Name
	Text     string      `xml:",chardata"`
	Children []plistNode `xml:",any"`
}

func (n plistNode) value(key string) plistNode {
	for i := 0; i+1 < len(n.Children); i++ {
		if n.Children[i].XMLName.Local == "key" && n.Children[i].Text == key {
			return n.Children[i+1]
		}
	}
	return plistNode{}
}

func parsePlist(t *testing.T, data []byte) plistNode {
	t.Helper()
	var p plistNode
	if err := xml.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Children) != 1 || p.Children[0].XMLName.Local != "dict" {
		t.Fatal("expected plist root dictionary")
	}
	return p.Children[0]
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeFile(t *testing.T, name string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(name, data, mode); err != nil {
		t.Fatal(err)
	}
}

func TestQueriesArePassedLiterallyToHandlers(t *testing.T) {
	root := parsePlist(t, readFile(t, "../../workflow/info.plist"))
	wantHandlers := map[string][]string{
		"n": {"search"}, "nst": {"finance"},
		"nm": {"map", "doNotUseIP"}, "nmi": {"map", "useIP"},
	}
	queries := []string{
		"$100", "$(printf substituted)", "`printf substituted`",
		"따옴표 \" ' \\ & ; /\n두 번째 줄", "workflow:update",
	}
	dir := t.TempDir()
	// Capture exactly the argv that the real run shim would pass to Go.
	writeFile(t, filepath.Join(dir, "run"), []byte("#!/bin/sh\nprintf '%s\\000' \"$@\"\n"), 0o755)
	for _, obj := range root.value("objects").Children {
		config := obj.value("config")
		keyword := config.value("keyword").Text
		handler, ok := wantHandlers[keyword]
		if !ok {
			continue
		}
		delete(wantHandlers, keyword)
		t.Run(keyword, func(t *testing.T) {
			if config.value("scriptargtype").Text != "1" {
				t.Fatal("filter must pass query as argv rather than interpolate shell source")
			}
			for _, query := range queries {
				cmd := exec.Command("/bin/bash", "-c", config.value("script").Text, "alfred", query)
				cmd.Dir = dir
				cmd.Env = []string{"PATH=/usr/bin:/bin"}
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("query %q: %v: %s", query, err, out)
				}
				got := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
				want := append(slices.Clone(handler), query)
				if !slices.Equal(got, want) {
					t.Errorf("query %q: argv = %q, want %q", query, got, want)
				}
			}
		})
	}
	if len(wantHandlers) != 0 {
		t.Errorf("missing filters: %v", wantHandlers)
	}
}

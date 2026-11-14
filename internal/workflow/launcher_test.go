package workflow_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLauncherPreparesOnlyNewOrChangedBinary(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "naversearch")
	log := filepath.Join(root, "preparations")
	tools := filepath.Join(root, "tools")
	if err := os.Mkdir(tools, 0700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "run"), readFile(t, "../../workflow/run"), 0755)
	writeFile(t, filepath.Join(tools, "xattr"), []byte("#!/bin/sh\nprintf 'xattr\\n' >> \"$PREP_LOG\"\n[ \"$FAIL_PREP\" != 1 ]\n"), 0755)
	program := []byte("#!/bin/sh\nprintf '%s\\000' \"$@\"\n")
	writeFile(t, bin, program, 0644)
	run := func(fail bool) {
		t.Helper()
		cmd := exec.Command("/bin/bash", filepath.Join(root, "run"), "search", "$100  *")
		cmd.Dir = t.TempDir()
		cmd.Env = append(os.Environ(), "PATH="+tools+":/usr/bin:/bin", "PREP_LOG="+log, "FAIL_PREP=0")
		if fail {
			cmd.Env = append(cmd.Env, "FAIL_PREP=1")
		}
		out, err := cmd.CombinedOutput()
		if err != nil || string(out) != "search\x00$100  *\x00" {
			t.Fatalf("launch: %q, %v", out, err)
		}
	}
	count := func(want int) {
		t.Helper()
		if got := strings.Count(string(readFile(t, log)), "xattr\n"); got != want {
			t.Fatalf("preparations=%d, want %d", got, want)
		}
	}
	run(false)
	run(false)
	count(1)
	// A replacement retaining an older mtime must still be prepared.
	replacement := filepath.Join(root, "replacement")
	writeFile(t, replacement, program, 0644)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(replacement, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, bin); err != nil {
		t.Fatal(err)
	}
	run(false)
	run(false)
	count(2)
	// In-place rebuild: inode stays the same, timestamp changes.
	stamp := filepath.Join(root, ".naversearch-ready/stamp")
	if err := os.Chtimes(stamp, old, old); err != nil {
		t.Fatal(err)
	}
	writeFile(t, bin, program, 0755)
	run(false)
	run(false)
	count(3)
	// Failed preparation must be retried, never recorded as successful.
	if err := os.Remove(stamp); err != nil {
		t.Fatal(err)
	}
	run(true)
	run(false)
	run(false)
	count(5)
	// A removed executable bit is repaired even on a prepared inode.
	if err := os.Chmod(bin, 0644); err != nil {
		t.Fatal(err)
	}
	run(false)
	run(false)
	count(6)
}

func TestPassThroughActionsPreserveLiteralQueries(t *testing.T) {
	root := parsePlist(t, readFile(t, "../../workflow/info.plist"))
	count := 0
	for _, obj := range root.value("objects").Children {
		if obj.value("type").Text != "alfred.workflow.action.script" {
			continue
		}
		config := obj.value("config")
		script := config.value("script").Text
		if !strings.HasPrefix(script, "printf") {
			continue
		}
		count++
		if config.value("scriptargtype").Text != "1" {
			t.Fatal("pass-through must use argv")
		}
		for _, query := range []string{"two  spaces", "*", "-n", "$100", "a\nb"} {
			cmd := exec.Command("/bin/bash", "-c", script, "alfred", query)
			out, err := cmd.CombinedOutput()
			if err != nil || string(out) != query {
				t.Fatalf("query %q became %q: %v", query, out, err)
			}
		}
	}
	if count != 3 {
		t.Fatalf("pass-through scripts=%d, want 3", count)
	}
}

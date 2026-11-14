package workflow_test

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackagingVersionsAndSourcePreservation(t *testing.T) {
	if _, err := exec.LookPath("zip"); err != nil {
		t.Skip("packaging requires zip")
	}
	root := t.TempDir()
	for _, dir := range []string{"workflow", "bin"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(root, "make.sh"), readFile(t, "../../make.sh"), 0o755)
	original := readFile(t, "../../workflow/info.plist")
	source := filepath.Join(root, "workflow/info.plist")
	writeFile(t, source, original, 0o644)
	// Exercise the actual packaging script without rebuilding Mach-O binaries
	// or creating Git commits/tags. Only the build and tag lookup are stubbed.
	writeFile(t, filepath.Join(root, "build.sh"), []byte("#!/bin/sh\nprintf binary > workflow/naversearch\n"), 0o755)
	writeFile(t, filepath.Join(root, "bin/git"), []byte("#!/bin/sh\n[ -n \"$TEST_TAG\" ] || exit 1\nprintf '%s\\n' \"$TEST_TAG\"\n"), 0o755)
	baseEnv := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "PATH" && key != "GITHUB_REF" && key != "WORKFLOW_VERSION" && key != "TEST_TAG" {
			baseEnv = append(baseEnv, entry)
		}
	}
	baseEnv = append(baseEnv, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	packagePath := filepath.Join(root, "NaverSearchPlus.alfredworkflow")
	tests := []struct {
		name, tag, ref, explicit, want string
	}{
		{"local tag", "v2.0.3", "", "", "2.0.3"},
		{"CI tag on second build", "v2.0.3", "refs/tags/v2.0.4", "", "2.0.4"},
		{"branch build", "v2.0.3", "refs/heads/main", "", "2.0.3"},
		{"explicit source archive version", "", "", "2.0.5", "2.0.5"},
		{"explicit override", "v2.0.3", "refs/tags/v2.0.4", "v2.0.6", "2.0.6"},
		{"missing version", "", "", "", ""},
		{"invalid version", "", "", "not-a-version", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous, _ := os.ReadFile(packagePath)
			cmd := exec.Command("/bin/sh", filepath.Join(root, "make.sh"))
			cmd.Dir = t.TempDir() // also prove invocation is independent of cwd
			cmd.Env = append(append([]string{}, baseEnv...), "TEST_TAG="+tt.tag, "GITHUB_REF="+tt.ref, "WORKFLOW_VERSION="+tt.explicit)
			out, err := cmd.CombinedOutput()
			if !bytes.Equal(readFile(t, source), original) {
				t.Fatal("packaging changed the source version template")
			}
			if tt.want == "" {
				if err == nil || !strings.Contains(string(out), "WORKFLOW_VERSION") {
					t.Fatalf("missing/invalid version must fail with instructions: %v: %s", err, out)
				}
				if !bytes.Equal(readFile(t, packagePath), previous) {
					t.Error("failed packaging replaced the previous archive")
				}
				return
			}
			if err != nil {
				t.Fatalf("packaging failed: %v: %s", err, out)
			}
			archive, err := zip.OpenReader(packagePath)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			file, err := archive.Open("info.plist")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				t.Fatal(err)
			}
			if got := parsePlist(t, data).value("version").Text; got != tt.want {
				t.Errorf("packaged version %q, want %q", got, tt.want)
			}
		})
	}
}

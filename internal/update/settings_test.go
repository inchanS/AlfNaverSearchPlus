package update

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

func TestAutoUpdateCommands(t *testing.T) {
	data := t.TempDir()
	t.Setenv("alfred_workflow_data", data)
	t.Setenv("alfred_workflow_cache", t.TempDir())
	if !autoUpdateEnabled() {
		t.Fatal("automatic checks must default to enabled")
	}
	for _, tc := range []struct {
		query string
		want  bool
	}{
		{"workflow:noautoupdate", false},
		{"workflow:noautoupdate", false},
		{"workflow:autoupdate", true},
	} {
		fb := alfred.New()
		if !HandleAutoUpdate(tc.query, fb) || autoUpdateEnabled() != tc.want {
			t.Fatalf("%s did not persist %v", tc.query, tc.want)
		}
		if len(fb.Items) != 1 || fb.Items[0].Valid {
			t.Fatalf("unexpected toggle feedback: %+v", fb.Items)
		}
		// Clearing/replacing cache must not reset the preference.
		t.Setenv("alfred_workflow_cache", t.TempDir())
		if autoUpdateEnabled() != tc.want {
			t.Fatal("preference depended on cache location")
		}
	}
	if HandleAutoUpdate("workflow:update", alfred.New()) || HandleAutoUpdate("ordinary query", alfred.New()) {
		t.Fatal("toggle handler intercepted a non-toggle query")
	}
	entries, err := os.ReadDir(data)
	if err != nil || len(entries) != 1 || entries[0].Name() != settingsFile {
		t.Fatalf("unexpected persistent files: %v, %v", entries, err)
	}
}

func TestDisabledChecksStillCleanCacheAndHideNotice(t *testing.T) {
	t.Setenv("alfred_workflow_data", t.TempDir())
	cacheDir := t.TempDir()
	t.Setenv("alfred_workflow_cache", cacheDir)
	t.Setenv("alfred_workflow_version", "2.0.5")
	if err := setAutoUpdate(false); err != nil {
		t.Fatal(err)
	}
	writeInfo(info{Version: "v3.2.0", Available: true})
	if err := cache.Write("nav_stale", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(cacheDir, "nav_stale.json")
	old := time.Now().Add(-2 * cacheMaxAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	u := stubHTTP(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("disabled automatic checks made a network request")
		return nil, nil
	})
	u.runCheck()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("disabled checks prevented cache cleanup: %v", err)
	}
	if _, ok := cache.Read(stampKey, 0); ok {
		t.Fatal("disabled check wrote a successful release-check stamp")
	}
	// Prevent spawning a real worker from MaybeShow during this unit test.
	if err := cache.Write(workerKey, []byte("fresh")); err != nil {
		t.Fatal(err)
	}
	fb := alfred.New()
	MaybeShow(fb)
	if len(fb.Items) != 0 {
		t.Fatal("disabled checks still displayed the cached update notice")
	}
	if err := setAutoUpdate(true); err != nil {
		t.Fatal(err)
	}
	MaybeShow(fb)
	if len(fb.Items) != 1 {
		t.Fatal("re-enabling did not restore update notices")
	}
}

func TestToggleReportsSaveFailure(t *testing.T) {
	// A regular file cannot be used as Alfred's workflow data directory.
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("alfred_workflow_data", file)
	fb := alfred.New()
	if !HandleAutoUpdate("workflow:noautoupdate", fb) || len(fb.Items) != 1 ||
		fb.Items[0].Title != "Could not save automatic update setting" {
		t.Fatalf("save failure reported as success: %+v", fb.Items)
	}
}

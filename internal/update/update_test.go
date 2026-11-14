package update

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubHTTP(t *testing.T, f testTransport) *updater {
	t.Helper()
	return &updater{http: httpx.New(&http.Client{Transport: f})}
}

func TestFailedCheckCanRetry(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	t.Setenv("alfred_workflow_version", "2.0.3")
	writeInfo(info{Version: "v2.0.2"})
	requests := 0
	u := stubHTTP(t, func(*http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return nil, errors.New("temporarily offline")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v2.0.4","assets":[]}`)),
		}, nil
	})
	u.runCheck()
	if _, ok := cache.Read(stampKey, 0); ok {
		t.Fatal("failed check was recorded as a successful weekly check")
	}
	if got, ok := readInfo(); !ok || got.Version != "v2.0.2" {
		t.Fatalf("failed check discarded previous release info: %+v", got)
	}
	u.runCheck()
	if requests != 2 {
		t.Fatalf("recovered network was not retried: %d requests", requests)
	}
	if got, ok := readInfo(); !ok || got.Version != "v2.0.4" {
		t.Fatalf("retry did not refresh release info: %+v", got)
	}
	if _, ok := cache.Read(stampKey, checkFrequency); !ok {
		t.Fatal("successful check was not recorded")
	}
	u.runCheck()
	if requests != 2 {
		t.Errorf("successful check did not retain weekly throttling: %d requests", requests)
	}
}

func TestInstallOnlyDownloadsNewerReleases(t *testing.T) {
	for _, offline := range []bool{false, true} {
		for _, version := range []string{"v2.0.2", "v2.0.3", "v2.0.4", ""} {
			name := "fresh/" + version
			if offline {
				name = "cached/" + version
			}
			t.Run(name, func(t *testing.T) {
				t.Setenv("alfred_workflow_cache", t.TempDir())
				t.Setenv("alfred_workflow_version", "2.0.3")
				const asset = "https://example.invalid/update.alfredworkflow"
				writeInfo(info{Available: true, Version: version, URL: asset})
				var downloads int
				u := stubHTTP(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.String() == asset {
						downloads++
						// Stop before writing an installer or invoking /usr/bin/open.
						return nil, errors.New("test download intentionally blocked")
					}
					if r.URL.Host != "api.github.com" {
						t.Fatalf("unexpected request: %s", r.URL)
					}
					if offline {
						return nil, errors.New("release API unavailable")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{"tag_name":"` + version +
							`","assets":[{"name":"NaverSearchPlus.alfredworkflow","browser_download_url":"` + asset + `"}]}`)),
					}, nil
				})
				fb := alfred.New()
				u.install(fb)
				wantDownloads, wantTitle := 0, "No update available"
				if version == "v2.0.4" {
					wantDownloads, wantTitle = 1, "Update download failed"
				}
				if downloads != wantDownloads {
					t.Errorf("version %q: %d downloads, want %d", version, downloads, wantDownloads)
				}
				if len(fb.Items) != 1 || fb.Items[0].Title != wantTitle {
					t.Errorf("unexpected feedback: %+v", fb.Items)
				}
			})
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		remote, current string
		want            bool
	}{
		{"v1.0.0", "0.3.1", true},
		{"v1.0.0", "1.0.0", false},
		{"v0.3.1", "1.0.0", false},
		{"v1.2.0", "v1.1.9", true},
		{"1.0.1", "1.0.0", true},
		{"v1.0.0", "", true},            // unknown current -> treat remote as newer
		{"", "1.0.0", false},            // no remote tag -> not newer
		{"v1.0.0-beta", "1.0.0", false}, // pre-release suffix ignored, equal
	}
	for _, c := range cases {
		if got := isNewer(c.remote, c.current); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.remote, c.current, got, c.want)
		}
	}
}

func TestShouldNotify(t *testing.T) {
	cases := []struct {
		name    string
		inf     info
		current string
		want    bool
	}{
		{"newer cached release", info{Version: "v1.1.0"}, "1.0.0", true},
		{"same version -> no notice", info{Version: "v1.0.0"}, "1.0.0", false},
		{"empty cached version", info{Version: ""}, "1.0.0", false},
		// Regression: a stale cache written on an older install (v0.2.1) must not
		// keep nagging after the user has updated to a newer version.
		{"stale older cached version", info{Version: "v0.2.1"}, "1.0.0", false},
		// The stored Available flag must not force a notice on its own; the
		// decision is recomputed from the versions.
		{"stale Available:true ignored", info{Available: true, Version: "v0.2.1"}, "1.0.0", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldNotify(c.inf, c.current); got != c.want {
				t.Errorf("shouldNotify(%+v, %q) = %v, want %v", c.inf, c.current, got, c.want)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("download interrupted") }

func TestDownloadUsesSeparateBudgetAndUniqueFiles(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	u := stubHTTP(t, func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining <= releaseTimeout || remaining > downloadTimeout {
			t.Fatalf("download budget: %s", remaining)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("archive"))}, nil
	})
	first, err := u.download("https://example.invalid/asset")
	if err != nil {
		t.Fatal(err)
	}
	second, err := u.download("https://example.invalid/asset")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !strings.HasSuffix(first, ".alfredworkflow") {
		t.Fatalf("download paths: %q, %q", first, second)
	}
	for _, path := range []string{first, second} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "archive" {
			t.Fatalf("downloaded data: %q, %v", data, err)
		}
	}
}

func TestInterruptedDownloadRemovesPartialFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	u := stubHTTP(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(io.MultiReader(strings.NewReader("partial"), failingReader{}))}, nil
	})
	if path, err := u.download("https://example.invalid/asset"); err == nil || path != "" {
		t.Fatalf("partial download accepted: %q, %v", path, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial files left: %v, %v", entries, err)
	}
}

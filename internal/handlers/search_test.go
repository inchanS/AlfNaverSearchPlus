package handlers

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

func TestSearchAndDictionaryResults(t *testing.T) {
	cases := []struct {
		name, body, path, param, title, copy, lang string
		run                                        func(*request, *alfred.Feedback, string) error
	}{
		{"search", `{"items":[[["hello"],[""]]]}`, "/nx/ac", "q", "Search Naver for 'hello'", "hello", "", (*request).search},
		{"shopping", `{"result":{"keywordList":[{"keywordName":"hello"},{"keywordName":""}]}}`, "/api/modules/gnb/auto-complete", "keyword", "Search Naver Shopping for 'hello'", "hello", "", (*request).shopping},
		{"terms", `{"items":[[["hello"],[""]]]}`, "/ac", "q", "hello", "hello", "", (*request).terms},
		{"English Korean", `{"items":[[[["hello"],[""],["안녕"]],[[""]]]]}`, "/enko/ac", "q", "hello     안녕", "안녕", "enko", (*request).endic},
		{"Korean", `{"items":[[[["hello"]],[[""]]]]}`, "/koko/ac", "q", "hello", "hello", "koko", (*request).krdic},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("alfred_workflow_cache", t.TempDir())
			calls := 0
			r := testRequest(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != tt.path || req.URL.Query().Get(tt.param) != "he" {
					t.Fatalf("request: %s", req.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tt.body))}, nil
			})
			for i := 0; i < 2; i++ {
				fb := alfred.New()
				if err := tt.run(r, fb, "he"); err != nil {
					t.Fatal(err)
				}
				if len(fb.Items) != 2 {
					t.Fatalf("items=%d", len(fb.Items))
				}
				head, item := fb.Items[0], fb.Items[1]
				if head.Arg != "he" || !head.Valid {
					t.Fatalf("fallback: %+v", head)
				}
				if item.Title != tt.title || item.Arg != "hello" || item.Autocomplete != "hello" || !item.Valid || item.Text == nil || item.Text.Copy != tt.copy || item.Text.LargeType != "hello" || item.QuicklookURL == "" {
					t.Fatalf("suggestion: %+v", item)
				}
				if tt.lang != "" && (head.Variables["lang"] != tt.lang || item.Variables["lang"] != tt.lang) {
					t.Fatal("dictionary route lost")
				}
			}
			if calls != 1 {
				t.Fatalf("requests=%d", calls)
			}
		})
	}
}

func TestSearchRecoversInvalidCacheAndResponse(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	if err := cache.Write(cache.Key("nav", "hello"), []byte("broken")); err != nil {
		t.Fatal(err)
	}
	calls := 0
	r := testRequest(t, func(*http.Request) (*http.Response, error) {
		calls++
		body := `{"items":[[["hello"]]]}`
		if calls == 1 {
			body = "<html>temporary failure</html>"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	fb := alfred.New()
	if err := r.search(fb, "hello"); err == nil || len(fb.Items) != 1 || !fb.Items[0].Valid {
		t.Fatal("invalid response lost fallback")
	}
	fb = alfred.New()
	if err := r.search(fb, "hello"); err != nil || len(fb.Items) != 2 {
		t.Fatalf("recovery: %v, %+v", err, fb.Items)
	}
	if calls != 2 {
		t.Fatalf("bad response cached: calls=%d", calls)
	}
}

func TestSearchSurvivesCacheWriteFailure(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("alfred_workflow_cache", filepath.Join(file, "cache"))
	r := testRequest(t, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"items":[[["hello"]]]}`))}, nil
	})
	fb := alfred.New()
	if err := r.search(fb, "hello"); err != nil || len(fb.Items) != 2 {
		t.Fatalf("lost result: %v, %+v", err, fb.Items)
	}
}

func TestSafePreservesFallbackOnErrorAndPanic(t *testing.T) {
	for _, panicBody := range []bool{false, true} {
		fb := alfred.New()
		err := safe(fb, func(fb *alfred.Feedback) error {
			fb.Add(alfred.ItemOpts{Title: "fallback", Valid: true})
			if panicBody {
				panic("failure")
			}
			return errors.New("failure")
		})
		if err == nil || len(fb.Items) != 1 || !fb.Items[0].Valid {
			t.Fatalf("recovery: %v, %+v", err, fb)
		}
	}
}

package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

type mapTestTransport func(*http.Request) (*http.Response, error)

func (f mapTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMapNavigationSurvivesAPIFailure(t *testing.T) {
	for _, useIP := range []bool{false, true} {
		for _, category := range []string{"", "place", "address", "bus"} {
			for _, failure := range []string{"network", "http", "json", "empty results"} {
				name := category + "/" + failure
				if useIP {
					name += "/IP"
				}
				t.Run(name, func(t *testing.T) {
					t.Setenv("alfred_workflow_cache", t.TempDir())
					t.Setenv("map_type", category)
					mode, state := "doNotUseIP", `{"use":false}`
					if useIP {
						mode, state = "useIP", `{"use":true}`
					}
					for key, data := range map[string]string{
						"use_ip": state, "location_data": `{"lat":37.5665,"lng":126.9780}`,
					} {
						if err := cache.Write(key, []byte(data)); err != nil {
							t.Fatal(err)
						}
					}
					original := http.DefaultTransport
					t.Cleanup(func() { http.DefaultTransport = original })
					http.DefaultTransport = mapTestTransport(func(*http.Request) (*http.Response, error) {
						if failure == "network" {
							return nil, errors.New("offline")
						}
						status, body := http.StatusOK, `{"all":[],"place":[],"address":[],"bus":[]}`
						if failure == "http" {
							status = http.StatusServiceUnavailable
						} else if failure == "json" {
							body = "invalid JSON"
						}
						return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
					})

					const word = "서울시청"
					fb := alfred.New()
					var err error
					if category == "" {
						err = mapSearch(fb, mode, word)
					} else {
						err = mapHub(fb, word)
					}
					if (err != nil) != (failure != "empty results") {
						t.Fatalf("unexpected error: %v", err)
					}
					wantCount := 1
					if category != "" {
						wantCount++
					}
					if failure == "empty results" {
						wantCount++
					}
					if len(fb.Items) != wantCount {
						t.Fatalf("got %d items, want %d", len(fb.Items), wantCount)
					}
					search := fb.Items[0]
					wantURL := "https://map.naver.com/p/search/%EC%84%9C%EC%9A%B8%EC%8B%9C%EC%B2%AD"
					if !search.Valid || search.Arg != wantURL || search.QuicklookURL != wantURL {
						t.Errorf("web search unavailable: %+v", search)
					}
					if category != "" {
						ret := fb.Items[1]
						if !ret.Valid || ret.Arg != word || ret.Variables["return"] != true || ret.Variables["useIP"] != useIP {
							t.Errorf("return navigation lost: %+v", ret)
						}
					}
				})
			}
		}
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

func mapResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
}

func TestMapOverviewAndCategoriesShareResponse(t *testing.T) {
	for _, useIP := range []bool{false, true} {
		t.Run(strconv.FormatBool(useIP), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("alfred_workflow_cache", dir)
			t.Setenv("useIP", strconv.FormatBool(useIP))
			if err := cache.Write("location_data", []byte(`{"lat":37.5,"lng":127.0}`)); err != nil {
				t.Fatal(err)
			}
			place := map[string]any{"title": "카페", "roadAddress": "도로명 주소", "jibunAddress": "지번 주소", "type": "place", "id": json.Number("9007199254740993")}
			address := map[string]any{"fullAddress": "서울 주소", "title": "주소", "y": json.Number("37.5"), "x": json.Number("127.0")}
			bus := map[string]any{"title": "100", "cityName": "서울", "id": "123"}
			body, err := json.Marshal(map[string]any{"place": []any{place}, "address": []any{address}, "bus": []any{bus}, "all": []any{map[string]any{"place": place}, map[string]any{"address": address}, map[string]any{"bus": bus}}})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			r := testRequest(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/p/api/search/instant-search" {
					t.Fatalf("unexpected request: %s", req.URL)
				}
				return mapResponse(string(body)), nil
			})
			mode := "doNotUseIP"
			if useIP {
				mode = "useIP"
			}
			overview := alfred.New()
			if err := r.mapSearch(overview, mode, "카페"); err != nil {
				t.Fatal(err)
			}
			if len(overview.Items) != 7 {
				t.Fatalf("overview items=%d", len(overview.Items))
			}
			for i, cat := range []string{"place", "address", "bus"} {
				item := overview.Items[i+1]
				if item.Variables["map_type"] != cat || item.Variables["useIP"] != useIP {
					t.Fatalf("category context: %+v", item)
				}
				t.Setenv("map_type", cat)
				fb := alfred.New()
				if err := r.mapHub(fb, "카페"); err != nil {
					t.Fatal(err)
				}
				if len(fb.Items) != 3 || fb.Items[1].Variables["useIP"] != useIP {
					t.Fatalf("category rows: %+v", fb.Items)
				}
				got, original := fb.Items[2], overview.Items[i+4]
				if got.Arg != original.Arg || got.QuicklookURL != got.Arg || !got.Valid || got.Icon == nil || got.Text == nil {
					t.Fatalf("rendered result: %+v", got)
				}
				switch cat {
				case "place":
					if !strings.HasSuffix(got.Arg, "/place/9007199254740993") || got.Subtitle != "도로명 주소" {
						t.Fatalf("place: %+v", got)
					}
				case "address":
					if !strings.Contains(got.Arg, "/entry/address/37.5,127.0,") {
						t.Fatalf("address: %+v", got)
					}
				case "bus":
					if !strings.HasSuffix(got.Arg, "/bus-route/123") || got.Subtitle != "서울버스 100" {
						t.Fatalf("bus: %+v", got)
					}
				}
			}
			if calls != 1 {
				t.Fatalf("overview and three categories made %d requests", calls)
			}
			if _, err := os.Stat(filepath.Join(dir, "use_ip.json")); !os.IsNotExist(err) {
				t.Fatalf("shared mode file was written: %v", err)
			}
		})
	}
}

func TestMapCacheChangesWithCoordinatesAndMode(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	t.Setenv("latitude", "37.5")
	t.Setenv("longitude", "127.0")
	calls := 0
	r := testRequest(t, func(req *http.Request) (*http.Response, error) {
		calls++
		return mapResponse(fmt.Sprintf(`{"place":[{}],"all":[{"place":{"title":%q,"type":"place","id":"1"}}]}`, req.URL.Query().Get("coords"))), nil
	})
	for _, lat := range []string{"37.5", "35.1"} {
		t.Setenv("latitude", lat)
		fb := alfred.New()
		if err := r.mapSearch(fb, "doNotUseIP", "cafe"); err != nil {
			t.Fatal(err)
		}
		if got := fb.Items[len(fb.Items)-1].Autocomplete; got != lat+",127.0" {
			t.Fatalf("stale coordinates: %q", got)
		}
	}
	if calls != 2 {
		t.Fatalf("coordinate requests=%d", calls)
	}
	if err := cache.Write("location_data", []byte(`{"lat":"37.5","lng":"127.0"}`)); err != nil {
		t.Fatal(err)
	}
	// An unrelated configured-location search cannot change an IP category's mode.
	t.Setenv("map_type", "place")
	t.Setenv("useIP", "true")
	if err := cache.Write("use_ip", []byte(`{"use":false}`)); err != nil {
		t.Fatal(err)
	}
	fb := alfred.New()
	if err := r.mapHub(fb, "cafe"); err != nil {
		t.Fatal(err)
	}
	if calls != 3 || fb.Items[1].Variables["useIP"] != true {
		t.Fatalf("mode context lost: calls=%d, %+v", calls, fb.Items[1])
	}
}

func TestMapSharesOneDeadline(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	r := testRequest(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			time.Sleep(30 * time.Millisecond)
			return mapResponse(`{"lngLat":{"lat":37.5,"lng":127.0}}`), nil
		}
		want, _ := ctx.Deadline()
		got, _ := req.Context().Deadline()
		if !got.Equal(want) {
			t.Fatalf("search reset the deadline: %s != %s", got, want)
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	r.ctx = ctx
	fb := alfred.New()
	if err := r.mapSearch(fb, "useIP", "cafe"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: %v", err)
	}
	if calls != 2 || len(fb.Items) != 1 || !fb.Items[0].Valid {
		t.Fatal("deadline lost fallback")
	}
}

func TestInvalidIPLocationFallsBackWithoutCaching(t *testing.T) {
	for _, body := range []string{`{"lngLat":null}`, `{"lngLat":{"lat":"NaN","lng":127}}`, `{"lngLat":{"lat":100,"lng":127}}`} {
		t.Run(body, func(t *testing.T) {
			t.Setenv("alfred_workflow_cache", t.TempDir())
			t.Setenv("latitude", "37.5")
			t.Setenv("longitude", "127.0")
			r := testRequest(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/p/api/location" {
					deadline, ok := req.Context().Deadline()
					if !ok || time.Until(deadline) > locationTimeout {
						t.Fatal("missing location budget")
					}
					return mapResponse(body), nil
				}
				if req.URL.Query().Get("coords") != "37.5,127.0" {
					t.Fatal("default location lost")
				}
				return mapResponse(`{"all":[],"place":[],"address":[],"bus":[]}`), nil
			})
			if err := r.mapSearch(alfred.New(), "useIP", "cafe"); err != nil {
				t.Fatal(err)
			}
			if _, ok := cache.Read("location_data", 0); ok {
				t.Fatal("invalid location cached")
			}
		})
	}
}

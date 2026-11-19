package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

const mapUIFixture = `{"place":[{"title":"카페 {file:secret} & 'coffee'","roadAddress":"서울 도로 1","type":"place","id":"123"}],"address":[{"fullAddress":"서울 주소 2","title":"지번","x":"127","y":"37"}],"bus":[{"title":"100","cityName":"서울","id":"456"}],"all":[{"bus":{"title":"100","cityName":"서울","id":"456"}},{"place":{"title":"카페 {file:secret} & 'coffee'","roadAddress":"서울 도로 1","type":"place","id":"123"}}]}`

func uiRows(out uiObject, q mapUIQuery) []uiObject {
	return out["variables"].(uiObject)[q.id()+"_rows"].([]uiObject)
}

func TestMapUIReusesResponseAcrossCategories(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	calls := 0
	r := testRequest(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Query().Get("type") != "all" || req.URL.Query().Get("query") != "카페" {
			t.Fatal(req.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(mapUIFixture))}, nil
	})
	for _, category := range []string{"all", "place", "address", "bus"} {
		q := newMapUIQuery([]string{"카페", "doNotUseIP", category})
		out := r.searchMapUI(q)
		rows := uiRows(out, q)
		want := 1
		if category == "all" {
			want = 2
		}
		if len(rows) != want {
			t.Fatalf("%s rows: %v", category, rows)
		}
		switch category {
		case "all":
			if rows[0]["name"] != "100" {
				t.Fatal("API ordering changed")
			}
		case "place":
			if rows[0]["name"] != "카페 {file:secret} & 'coffee'" || rows[0]["address"] != "서울 도로 1" {
				t.Fatal(rows)
			}
			if !strings.HasSuffix(rows[0]["url"].(string), "/place/123") {
				t.Fatal(rows)
			}
		case "address":
			if rows[0]["url"] != "https://map.naver.com/p/entry/address/37,127,%EC%84%9C%EC%9A%B8%20%EC%A3%BC%EC%86%8C%202" {
				t.Fatal(rows)
			}
		case "bus":
			if rows[0]["hasaddress"] != false || rows[0]["address"] != "" {
				t.Fatal(rows)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("category changes made %d API calls", calls)
	}
}

func TestMapUIFallbackAndErrorsKeepWebSearch(t *testing.T) {
	for _, failure := range []string{"ip", "search", "json", "empty"} {
		t.Run(failure, func(t *testing.T) {
			t.Setenv("alfred_workflow_cache", t.TempDir())
			t.Setenv("latitude", "35")
			t.Setenv("longitude", "129")
			r := testRequest(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/p/api/location" {
					return nil, errors.New("offline")
				}
				if req.URL.Query().Get("coords") != "35,129" {
					t.Fatal("configured fallback coordinates lost")
				}
				if failure == "search" {
					return nil, errors.New("offline")
				}
				body := mapUIFixture
				if failure == "json" {
					body = "invalid JSON"
				}
				if failure == "empty" {
					body = `{"all":[],"place":[],"address":[],"bus":[]}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			q := newMapUIQuery([]string{"카페", "useIP", "all"})
			out := r.searchMapUI(q)
			b, _ := json.Marshal(out)
			s := string(b)
			if !strings.Contains(s, "IP 위치 조회 실패") || !strings.Contains(s, "네이버 지도에서 검색") || !strings.Contains(s, "https://map.naver.com/p/search/") {
				t.Fatal(s)
			}
			if failure != "ip" && len(uiRows(out, q)) != 0 {
				t.Fatal("failure retained actionable results")
			}
		})
	}
}

func TestMapUIEmptyQueryDoesNotFetch(t *testing.T) {
	r := testRequest(t, func(*http.Request) (*http.Response, error) { t.Fatal("empty query requested network"); return nil, nil })
	q := newMapUIQuery([]string{"  ", "useIP", "all"})
	if len(uiRows(r.searchMapUI(q), q)) != 0 {
		t.Fatal("empty query results")
	}
}

func TestMapUILateResponsesTargetOnlyMatchingInputs(t *testing.T) {
	queries := []mapUIQuery{
		newMapUIQuery([]string{"서울", "doNotUseIP", "all"}),
		newMapUIQuery([]string{"부산", "doNotUseIP", "all"}),
		newMapUIQuery([]string{"서울", "useIP", "all"}),
		newMapUIQuery([]string{"서울", "doNotUseIP", "place"}),
	}
	seen := map[string]bool{}
	for _, q := range queries {
		id := q.id()
		if seen[id] {
			t.Fatal("different searches share update target")
		}
		seen[id] = true
		prepare := q.prepare()["update"].(uiObject)["ids"].(uiObject)["map_content"].(uiObject)
		if prepare["components"].([]uiObject)[0]["id"] != id {
			t.Fatal("prepare target mismatch")
		}
		response := q.response(nil, "done", "empty")["update"].(uiObject)["ids"].(uiObject)
		if len(response) != 1 || response[id] == nil {
			t.Fatal("response can update another request's controls")
		}
	}
}

func TestMapResultAddressFallback(t *testing.T) {
	r := decodeMapResult("place", map[string]any{"title": "카페", "jibunAddress": "지번 주소"})
	if r.Address != "지번 주소" || r.Detail != r.Address {
		t.Fatal(r)
	}
}

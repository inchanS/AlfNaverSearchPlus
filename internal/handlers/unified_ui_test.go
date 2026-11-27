package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

func TestUnifiedShoppingUsesSelectedSuggestionAndCurrentView(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	const word = "workflow:update"
	if err := cache.Write(cache.Key("navs", word), []byte(`{"result":{"keywordList":[{"keywordName":"서울 & 우유"}]}}`)); err != nil {
		t.Fatal(err)
	}
	id := func() string {
		return prepareSearch("shopping", word, "shopping")["variables"].(uiJSON)["lookup"].(string)
	}
	first, next := id(), id()
	if first == next {
		t.Fatal("retyping the same query must invalidate its previous result view")
	}
	action := testRequest(t, nil).unifiedResults("shopping", word, first, "")
	ids := action["update"].(uiJSON)["ids"].(uiJSON)
	if len(ids) != 1 || ids["response_slot"] == nil {
		t.Fatal("async responses must go through the current request slot")
	}
	rows := resultRows(t, action, first)
	if len(rows) != 2 || rows[1]["name.text"] != "서울 & 우유" {
		t.Fatalf("missing suggestion: %v", rows)
	}
	reply := action["variables"].(uiJSON)["replacements"].(uiJSON)[first].(uiJSON)
	if reply["lookup"] != first || reply["url1"] != rows[0]["url"] || reply["url2"] != rows[1]["url"] || reply["url3"] != nil {
		t.Fatalf("numbered shortcuts do not match displayed rows: %v", reply)
	}
	const encoded = "%EC%84%9C%EC%9A%B8%20%26%20%EC%9A%B0%EC%9C%A0"
	for key, want := range map[string]string{
		"url":      "https://search.shopping.naver.com/search/all?query=" + encoded,
		"shiftURL": "https://search.shopping.naver.com/ns/search?query=" + encoded,
	} {
		if got := rows[1][key].(string); got != want || openResult(got, rows[1]["lookup"].(string), first)["open"] == nil {
			t.Errorf("%s = %q, want actionable %q", key, got, want)
		}
	}
}

func TestUnifiedEmptyQueryAndOfflineFallback(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	const id = "lookup_test"
	if got := testRequest(t, nil).unifiedResults("search", "  ", id, ""); len(got) != 0 {
		t.Fatalf("empty query started search: %v", got)
	}
	r := testRequest(t, func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	action := r.unifiedResults("search", "서울", id, "")
	rows := resultRows(t, action, id)
	if len(rows) != 1 || rows[0]["url"] == "" {
		t.Fatal("keep direct search when suggestions are unavailable")
	}
	if action := openResult("", id, id); len(action) != 0 {
		t.Fatal("an empty selection should leave the window open")
	}
}

func resultRows(t *testing.T, action uiJSON, lookup string) []uiJSON {
	t.Helper()
	replacements := action["variables"].(uiJSON)["replacements"].(uiJSON)
	entry, ok := replacements[lookup].(uiJSON)
	if !ok || len(replacements) != 1 {
		t.Fatalf("response must match only %q", lookup)
	}
	ids := entry["patch"].(uiJSON)["ids"].(uiJSON)
	return ids["results"].(uiJSON)["data"].(uiJSON)["src"].([]uiJSON)
}

func TestPreparedDirectSearchIsImmediatelyActionable(t *testing.T) {
	const word = "서울 & 역"
	for _, category := range []string{"search", "map", "shopping"} {
		t.Run(category, func(t *testing.T) {
			prepared := prepareSearch(category, " "+word+" ", category)
			id := prepared["variables"].(uiJSON)["lookup"].(string)
			ids := prepared["update"].(uiJSON)["ids"].(uiJSON)
			view := ids["results"].(uiJSON)
			rows := view["data"].(uiJSON)["src"].([]uiJSON)
			slot := ids["response_slot"].(uiJSON)["data"].(uiJSON)["src"].([]uiJSON)[0]
			if len(rows) != 1 || rows[0]["completion"] != word || rows[0]["uid"] != view["select"] || rows[0]["lookup"] != id ||
				slot["uid"] != id || slot["lookup"] != id || slot["url1"] != rows[0]["url"] || slot["url2"] != nil {
				t.Fatalf("unexpected immediate selection: %v", ids)
			}
			for _, field := range []string{"url", "shiftURL"} {
				dest := rows[0][field].(string)
				if openResult(dest, id, id)["open"] == nil || len(openResult(dest, "lookup_previous", id)) != 0 {
					t.Fatalf("incorrect lookup handling for %s", dest)
				}
			}
		})
	}
	for _, tc := range []struct{ category, query string }{{"map", "  "}, {"finance", word}} {
		ids := prepareSearch(tc.category, tc.query, "search")["update"].(uiJSON)["ids"].(uiJSON)
		if rows := ids["results"].(uiJSON)["data"].(uiJSON)["src"].([]uiJSON); len(rows) != 0 {
			t.Fatalf("unexpected immediate results for %+v: %v", tc, rows)
		}
	}
}

func TestUnifiedMapLocation(t *testing.T) {
	for _, tc := range []struct{ mode, pending, title, coords string }{
		{"useIP", "위치 확인 중…", "설정 위치 기준", "37.5,127.0"},
		{"", "위치 확인 중…", "IP 주소 기준", "35.1,129.1"},
		{"doNotUseIP", "설정 위치 기준", "설정 위치 기준", "37.5,127.0"},
		{"useIP", "위치 확인 중…", "IP 주소 기준", "35.1,129.1"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			t.Setenv("alfred_workflow_cache", t.TempDir())
			t.Setenv("nn_map_location", tc.mode)
			t.Setenv("latitude", "37.5")
			t.Setenv("longitude", "127.0")
			prepared := prepareSearch("map", "카페", "search")
			ids := prepared["update"].(uiJSON)["ids"].(uiJSON)
			if title := ids["map_location"].(uiJSON)["text"]; title != tc.pending {
				t.Fatalf("map label = %q, want %q", title, tc.pending)
			}
			var coords string
			r := testRequest(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/p/api/location" {
					if tc.mode == "doNotUseIP" {
						t.Fatal("configured coordinates must skip IP lookup")
					}
					if tc.title == "설정 위치 기준" {
						return nil, errors.New("IP lookup failed")
					}
					return mapResponse(`{"lngLat":{"lat":35.1,"lng":129.1}}`), nil
				}
				coords = req.URL.Query().Get("coords")
				return mapResponse("{}"), nil
			})
			action := r.unifiedResults("map", "카페", "lookup_test", "")
			entry := action["variables"].(uiJSON)["replacements"].(uiJSON)["lookup_test"].(uiJSON)
			label := entry["patch"].(uiJSON)["ids"].(uiJSON)["map_location"].(uiJSON)["text"]
			if label != tc.title {
				t.Fatalf("location label = %v, want %s", label, tc.title)
			}
			if coords != tc.coords {
				t.Fatalf("map search coordinates = %q, want %q", coords, tc.coords)
			}
		})
	}
}

func TestResumeMapByDestination(t *testing.T) {
	t.Setenv("nn_resume", "1")
	t.Setenv("alfred_workflow_data", t.TempDir())
	t.Setenv("alfred_workflow_cache", t.TempDir())
	t.Setenv("nn_map_location", "doNotUseIP")
	const query = "목욕탕"
	const lookup = "lookup_resume"
	r := testRequest(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("query") != query {
			t.Fatal("searched for the preview instead of the original query")
		}
		return mapResponse(`{"all":[{"place":{"type":"place","id":"22","title":"새 목욕탕"}},{"place":{"type":"place","id":"11","title":"우리 목욕탕"}}]}`), nil
	})
	rows := resultRows(t, r.unifiedResults("map", query, lookup, ""), lookup)
	s := lastSearch{Category: "map", Query: query, Selected: rows[2]["url"].(string)}
	if err := s.save(); err != nil {
		t.Fatal(err)
	}
	resumed := resumeSearch("")["variables"].(uiJSON)
	if resumed["source_query"] != query || resumed["category"] != "map" || resumed["restore_url"] != s.Selected {
		t.Fatalf("restored %v", resumed)
	}
	for _, tc := range []struct{ url, want string }{{s.Selected, "restored_result"}, {"missing", ""}} {
		action := r.unifiedResults("map", query, lookup, tc.url)
		patch := action["variables"].(uiJSON)["replacements"].(uiJSON)[lookup].(uiJSON)["patch"].(uiJSON)
		view := patch["ids"].(uiJSON)["results"].(uiJSON)
		if view["select"] != tc.want {
			t.Fatalf("selection = %v", view["select"])
		}
		if tc.want != "" && resultRows(t, action, lookup)[2]["uid"] != tc.want {
			t.Fatal("wrong place selected")
		}
	}
	if len(resumeSearch("새 검색")) != 0 {
		t.Fatal("explicit query was overwritten")
	}
	t.Setenv("nn_resume", "0")
	if len(resumeSearch("")) != 0 {
		t.Fatal("restored while disabled")
	}
	if err := (lastSearch{Category: "search", Query: "disabled"}).save(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("nn_resume", "1")
	if resumeSearch("")["variables"].(uiJSON)["source_query"] != query {
		t.Fatal("saved while disabled")
	}
	if err := (lastSearch{Category: "map"}).save(); err != nil {
		t.Fatal(err)
	}
	if len(resumeSearch("")) != 0 {
		t.Fatal("empty query did not clear the session")
	}
}

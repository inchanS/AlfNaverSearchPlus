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

func TestTypingRetainsVisibleRowsButInvalidatesTheirActions(t *testing.T) {
	for _, tc := range []struct {
		category, previous, word string
		clear                    bool
	}{
		{"search", "search", "서울역", false},
		{"search", "search", "", true},
		{"map", "search", "서울역", true},
	} {
		action := prepareSearch(tc.category, tc.word, tc.previous)
		ids := action["update"].(uiJSON)["ids"].(uiJSON)
		view := ids["results"].(uiJSON)
		_, cleared := view["data"]
		if cleared != tc.clear || view["components"] != nil {
			t.Fatalf("%+v: unexpected list update: %v", tc, view)
		}
		current := action["variables"].(uiJSON)["lookup"].(string)
		if got := openResult("https://map.naver.com", "lookup_previous", current); len(got) != 0 {
			t.Fatal("an old visible suggestion must not open while new results are pending")
		}
	}
}

func TestUnifiedMapLocation(t *testing.T) {
	for _, tc := range []struct{ mode, title, coords string }{
		{"", "IP 주소 기준", "35.1,129.1"},
		{"doNotUseIP", "설정 위치 기준", "37.5,127.0"},
		{"useIP", "IP 주소 기준", "35.1,129.1"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			t.Setenv("alfred_workflow_cache", t.TempDir())
			t.Setenv("nn_map_location", tc.mode)
			t.Setenv("latitude", "37.5")
			t.Setenv("longitude", "127.0")
			prepared := prepareSearch("map", "카페", "search")
			ids := prepared["update"].(uiJSON)["ids"].(uiJSON)
			if title := ids["map_location"].(uiJSON)["text"]; title != tc.title {
				t.Fatalf("map label = %q, want %q", title, tc.title)
			}
			var coords string
			r := testRequest(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/p/api/location" {
					if tc.mode == "doNotUseIP" {
						t.Fatal("configured coordinates must skip IP lookup")
					}
					return mapResponse(`{"lngLat":{"lat":35.1,"lng":129.1}}`), nil
				}
				coords = req.URL.Query().Get("coords")
				return mapResponse("{}"), nil
			})
			r.unifiedResults("map", "카페", "lookup_test", "")
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

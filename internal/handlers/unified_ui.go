package handlers

import (
	"crypto/rand"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
)

type uiJSON = map[string]any

var categoryNames = map[string]string{
	"search": "일반", "map": "지도", "finance": "증권", "shopping": "쇼핑",
}

func updateViews(ids uiJSON) uiJSON {
	return uiJSON{"update": uiJSON{"ids": ids}}
}

// Return typed JSON directly to the UI. Workflow stream variables cannot
// reliably carry the numeric types used by window and dataview constraints.
func unifiedLayout() uiJSON {
	rows := 6
	switch os.Getenv("nn_rows") {
	case "8":
		rows = 8
	case "10":
		rows = 10
	}
	font, secondary := "default", "small"
	rowHeight, chromeHeight := 43, 197
	switch os.Getenv("nn_font") {
	case "large":
		font, secondary = "large", "default"
		rowHeight, chromeHeight = 51, 211
	case "extra_large":
		font, secondary = "extralarge", "large"
		rowHeight, chromeHeight = 59, 227
	}
	listHeight := rowHeight * rows
	resize := uiJSON{"height": listHeight + chromeHeight}
	switch os.Getenv("nn_width") {
	case "theme":
		// Omitting width keeps the embedded window at Alfred's theme width.
	case "wide":
		resize["width"] = 760
	default:
		resize["width"] = 595
	}
	styles := uiJSON{"hint": uiJSON{"font": uiJSON{"size": secondary}}}
	for _, id := range []string{"query", "category_label", "tab_search", "tab_map", "tab_finance", "tab_shopping"} {
		styles[id] = uiJSON{"font": uiJSON{"size": font}}
	}
	return uiJSON{
		"variables": uiJSON{
			"nn_list_height": listHeight,
			"nn_row_height":  rowHeight, "nn_primary_font": font, "nn_secondary_font": secondary,
		},
		"update":  uiJSON{"ids": styles, "relayout": true},
		"replace": uiJSON{"ids": uiJSON{"results_area": "{file:results-area.json}"}},
		"window":  uiJSON{"resize": resize},
	}
}

func prepareSearch(category, word, previousCategory string) uiJSON {
	id := "lookup_" + rand.Text()
	// The hidden data provider has one slot. A reply can replace it only
	// while its uid is current. The visible DataView lives for the whole UI.
	slot := uiJSON{"uid": id, "patch": uiJSON{"ids": uiJSON{}}}
	ids := uiJSON{"response_slot": uiJSON{"data": uiJSON{"src": []uiJSON{slot}}}}
	empty := strings.TrimSpace(word) == ""
	results := uiJSON{"placeholder": "검색 중…"}
	if empty {
		results["placeholder"] = "검색어를 입력하세요."
	}
	if empty || category != previousCategory {
		results["data"] = uiJSON{"src": []uiJSON{}}
	}
	ids["results"] = results
	title := categoryNames[category] + " 검색"
	if category == "map" {
		title += " · 설정 위치 기준"
	}
	ids["category_label"] = uiJSON{"text": title}
	ids["hint"] = uiJSON{"text": keyboardHint(category)}
	action := updateViews(ids)
	action["variables"] = uiJSON{"category": category, "lookup": id}
	return action
}

func keyboardHint(category string) string {
	hint := "↩ 또는 ⌘↓ 목록으로    ↑↓ 선택    ↩ 열기    ⌘L 검색창"
	if category == "shopping" {
		hint += "    ⇧↩ 네이버플러스 스토어"
	}
	return hint
}

func (r *request) unifiedResults(category, word, id string) uiJSON {
	word = strings.TrimSpace(word)
	if word == "" || !strings.HasPrefix(id, "lookup_") {
		return uiJSON{}
	}
	fb := alfred.New()
	err := safe(fb, func(fb *alfred.Feedback) error {
		switch category {
		case "search":
			return r.search(fb, word)
		case "map":
			return r.mapSearch(fb, "doNotUseIP", word)
		case "shopping":
			return r.shopping(fb, word)
		default:
			return r.finance(fb, word)
		}
	})
	rows := []uiJSON{}
	for _, item := range fb.Items {
		// Map drill-down entries navigate between Script Filters. The unified
		// list shows the instant-search suggestions and their destinations.
		if item.Autocomplete == "" || item.Variables["map_type"] != nil {
			continue
		}
		destination := item.QuicklookURL
		alternate := destination
		if category == "shopping" {
			destination = quick("https://search.shopping.naver.com/search/all?query=%s", item.Arg)
		}
		if !item.Valid {
			destination, alternate = "", ""
		}
		detail := item.Subtitle
		if len(rows) == 0 && category != "finance" {
			detail = categoryNames[category] + "에서 입력한 검색어로 검색"
		}
		rows = append(rows, uiJSON{"name.text": item.Autocomplete, "detail.text": detail,
			"url": destination, "shiftURL": alternate, "lookup": id})
	}
	placeholder := "검색 결과가 없습니다."
	hint := keyboardHint(category)
	if err != nil {
		logf("nui %s: %v", category, err)
		placeholder = "자동완성을 불러오지 못했습니다."
		hint = "자동완성 조회 실패 · 카테고리를 다시 선택하면 재시도합니다."
	}
	patch := uiJSON{"ids": uiJSON{
		"results": uiJSON{"data": uiJSON{"src": rows}, "placeholder": placeholder},
		"hint":    uiJSON{"text": hint},
	}}
	// Matching is performed by Alfred at delivery time, including when a
	// slow request finishes after the same query was typed a second time.
	replacements := uiJSON{id: uiJSON{"uid": "done_" + id, "patch": patch}}
	action := updateViews(uiJSON{"response_slot": uiJSON{"data": uiJSON{"modify": uiJSON{
		"replace": uiJSON{"srcmap": "{var:replacements}"},
	}}}})
	action["variables"] = uiJSON{"replacements": replacements}
	return action
}

func openResult(destination, selectedLookup, currentLookup string) uiJSON {
	if selectedLookup == "" || selectedLookup != currentLookup {
		return uiJSON{}
	}
	u, err := url.Parse(destination)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return uiJSON{}
	}
	switch u.Host {
	case "search.naver.com", "search.shopping.naver.com", "map.naver.com", "stock.naver.com":
		return uiJSON{"variables": uiJSON{"destination": destination},
			"open": uiJSON{"type": "url", "src": "{var:destination}"}, "finish": uiJSON{}}
	}
	return uiJSON{}
}

func (r *request) unifiedUI(args []string) {
	var action = uiJSON{}
	command, category := at(args, 0), at(args, 1)
	if command == "layout" {
		action = unifiedLayout()
	} else if command == "open" {
		action = openResult(at(args, 1), at(args, 2), at(args, 3))
		// Restore before the JSON open action gives focus to the browser.
		if action["open"] != nil && os.Getenv("nnime_applied") != "" {
			if binary, err := os.Executable(); err == nil {
				helper := filepath.Join(filepath.Dir(binary), "input-source")
				cmd := exec.Command(helper, "leave", os.Getenv("nnime_previous"), os.Getenv("nnime_applied"))
				if err := cmd.Run(); err != nil {
					logf("nnime restore: %v", err)
				}
			}
			vars := action["variables"].(uiJSON)
			vars["nnime_previous"], vars["nnime_applied"] = "", ""
		}
	} else if _, ok := categoryNames[category]; ok {
		switch command {
		case "prepare":
			action = prepareSearch(category, at(args, 2), at(args, 3))
		case "search":
			action = r.unifiedResults(category, at(args, 2), at(args, 3))
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(action)
}

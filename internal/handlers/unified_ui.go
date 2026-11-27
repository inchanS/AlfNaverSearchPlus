package handlers

import (
	"crypto/rand"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/env"
	"github.com/inchanS/AlfNaverSearchPlus/internal/update"
)

type uiJSON = map[string]any

var categoryNames = map[string]string{
	"search": "일반", "map": "지도", "finance": "증권", "shopping": "쇼핑",
}

func updateViews(ids uiJSON) uiJSON {
	return uiJSON{"update": uiJSON{"ids": ids}}
}

func unifiedMapMode() string {
	return env.Str("nn_map_location", "useIP")
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
		results["placeholder"] = ""
	}
	if !empty && category != "finance" {
		row := directSearchRow(category, strings.TrimSpace(word), id)
		row["uid"], row["key_hint.text"] = "direct", "⌘1"
		results["data"] = uiJSON{"src": []uiJSON{row}}
		results["select"] = "direct"
		slot["lookup"], slot["url1"] = id, row["url"]
	} else if empty || category != previousCategory {
		results["data"] = uiJSON{"src": []uiJSON{}}
	}
	ids["results"] = results
	if category != previousCategory {
		if _, ok := categoryNames[previousCategory]; ok {
			ids["segment_"+previousCategory] = uiJSON{"background": "tabFill"}
			ids["tab_"+previousCategory] = uiJSON{"font": uiJSON{"size": "{var:nn_secondary_font}", "weight": "regular"}}
		}
		ids["segment_"+category] = uiJSON{"background": "activeTabFill"}
		ids["tab_"+category] = uiJSON{"font": uiJSON{"size": "{var:nn_secondary_font}", "weight": "semibold"}}
	}
	label := ""
	if category == "map" && !empty {
		label = "설정 위치 기준"
		if unifiedMapMode() == "useIP" {
			label = "위치 확인 중…"
		}
	}
	ids["map_location"] = uiJSON{"text": label}
	ids["hint"] = uiJSON{"text": keyboardHint(category)}
	action := updateViews(ids)
	action["variables"] = uiJSON{"category": category, "lookup": id, "source_query": word, "restore_url": ""}
	return action
}

// Direct destinations are available before autocomplete finishes.
func directSearchRow(category, word, id string) uiJSON {
	pattern := map[string]string{"search": searchURL, "map": mapSearchURL, "shopping": shoppingStoreURL}[category]
	return unifiedRow(category, word, categoryNames[category]+"에서 입력한 검색어로 검색", quick(pattern, word), id, true)
}

func unifiedRow(category, word, detail, quicklook, id string, valid bool) uiJSON {
	destination := quicklook
	if category == "shopping" {
		destination = quick(shoppingCompareURL, word)
	}
	if !valid {
		destination, quicklook = "", ""
	}
	return uiJSON{"name.text": word, "completion": word, "detail.text": detail,
		"url": destination, "shiftURL": quicklook, "lookup": id}
}

func keyboardHint(category string) string {
	hint := "↑↓ 선택   ↩ 열기   ⌘1–9 바로 열기   ⌘L 검색창"
	if category == "shopping" {
		hint += "    ⇧↩ 네이버플러스 스토어"
	}
	return hint
}

func (r *request) unifiedResults(category, word, id, selected string) uiJSON {
	word = strings.TrimSpace(word)
	if word == "" || !strings.HasPrefix(id, "lookup_") {
		return uiJSON{}
	}
	var loc location
	fb := alfred.New()
	err := safe(fb, func(fb *alfred.Feedback) error {
		switch category {
		case "search":
			return r.search(fb, word)
		case "map":
			useIP := unifiedMapMode() == "useIP"
			loc = r.getLocation(useIP)
			return r.mapSearchAt(fb, loc, useIP, word)
		case "shopping":
			return r.shopping(fb, word)
		default:
			return r.finance(fb, word)
		}
	})
	rows := []uiJSON{}
	selection := ""
	for _, item := range fb.Items {
		// Map drill-down entries navigate between Script Filters. The unified
		// list shows the instant-search suggestions and their destinations.
		if item.Autocomplete == "" || item.Variables["map_type"] != nil {
			continue
		}
		detail := item.Subtitle
		if len(rows) == 0 && category != "finance" {
			detail = categoryNames[category] + "에서 입력한 검색어로 검색"
		}
		row := unifiedRow(category, item.Autocomplete, detail, item.QuicklookURL, id, item.Valid)
		if selected != "" && row["url"] == selected && selection == "" {
			selection = "restored_result"
			row["uid"] = selection
		}
		rows = append(rows, row)
	}
	placeholder := "검색 결과가 없습니다."
	hint := keyboardHint(category)
	if err != nil {
		logf("nui %s: %v", category, err)
		placeholder = "자동완성을 불러오지 못했습니다."
		hint = "자동완성 조회 실패 · 카테고리를 다시 선택하면 재시도합니다."
	}
	results := uiJSON{"data": uiJSON{"src": rows}, "placeholder": placeholder}
	if selected != "" {
		results["select"] = selection
	}
	ids := uiJSON{
		"results": results,
		"hint":    uiJSON{"text": hint},
	}
	if category == "map" {
		label := "설정 위치 기준"
		if loc.fromIP {
			label = "IP 주소 기준"
		}
		ids["map_location"] = uiJSON{"text": label}
	}
	patch := uiJSON{"ids": ids}
	// Matching is performed by Alfred at delivery time, including when a
	// slow request finishes after the same query was typed a second time.
	reply := uiJSON{"uid": "done_" + id, "patch": patch, "lookup": id}
	for i, row := range rows[:min(len(rows), 9)] {
		number := strconv.Itoa(i + 1)
		if row["url"] != "" {
			row["key_hint.text"] = "⌘" + number
		}
		reply["url"+number] = row["url"]
	}
	replacements := uiJSON{id: reply}
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
	if command == "start" {
		update.MaybeSpawnWorker()
		initial := at(args, 1)
		if initial != "" {
			action = prepareSearch("search", initial, "search")
		} else {
			action = resumeSearch("")
		}
		if version := update.AvailableVersion(); version != "" {
			if action["update"] == nil {
				action = updateViews(uiJSON{})
			}
			ids := action["update"].(uiJSON)["ids"].(uiJSON)
			ids["service_name"] = uiJSON{"text": "업데이트 " + version + " · n workflow:update"}
		}
	} else if command == "resume" {
		action = resumeSearch(at(args, 1))
	} else if command == "remember" {
		selected := at(args, 3)
		if at(args, 4) != at(args, 5) {
			selected = ""
		}
		if err := (lastSearch{Category: category, Query: at(args, 2), Selected: selected}).save(); err != nil {
			logf("save last search: %v", err)
		}
	} else if command == "preview" {
		if completion, selected := at(args, 1), at(args, 2); completion != "" && selected != "" && selected == at(args, 3) {
			action = updateViews(uiJSON{"query": uiJSON{"text": "{var:completion}"}})
			action["variables"] = uiJSON{"completion": completion}
		}
	} else if command == "open" {
		action = openResult(at(args, 1), at(args, 2), at(args, 3))
		if action["open"] != nil && os.Getenv("nn_resume") == "1" {
			s := lastSearch{Category: at(args, 4), Query: at(args, 5), Selected: at(args, 6)}
			if err := s.save(); err != nil {
				logf("save last search: %v", err)
			} else {
				action["variables"].(uiJSON)["search_saved"] = "1"
			}
		}
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
			action = r.unifiedResults(category, at(args, 2), at(args, 3), at(args, 4))
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(action)
}

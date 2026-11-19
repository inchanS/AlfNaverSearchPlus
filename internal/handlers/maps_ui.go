package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
)

type uiObject = map[string]any

type mapUIQuery struct{ Word, Mode, Category string }

func newMapUIQuery(args []string) mapUIQuery {
	q := mapUIQuery{strings.TrimSpace(at(args, 0)), at(args, 1), at(args, 2)}
	if q.Mode != "useIP" {
		q.Mode = "doNotUseIP"
	}
	switch q.Category {
	case "place", "address", "bus":
	default:
		q.Category = "all"
	}
	return q
}

// The synchronous prepare action removes the old request's view before starting
// asynchronous work. A late response can only update its own input-specific ID;
// it cannot replace results for another query, location mode or category. This
// requires no persistent jobs, processes or query state on disk.
func (q mapUIQuery) id() string {
	b, _ := json.Marshal(q)
	return fmt.Sprintf("map_%x", sha256.Sum256(b))
}

func uiUpdate(id string, properties uiObject) uiObject {
	return uiObject{"update": uiObject{"relayout": true, "ids": uiObject{id: properties}}}
}

func uiLabel(text string) uiObject {
	return uiObject{"type": "label", "text": text, "colour": "secondaryLabelColor"}
}

func uiView(components ...uiObject) uiObject {
	return uiObject{"type": "view", "layout": "inline", "direction": "vertical", "spacing": 8, "components": components}
}

func uiButton(text string, action uiObject) uiObject {
	return uiObject{"type": "button", "text": text, "events": uiObject{"triggers": "actioned", "action": action}}
}

func uiJSONAction(src uiObject) uiObject { return uiObject{"type": "json", "src": src} }

func (q mapUIQuery) webButton() uiObject {
	b := uiButton("네이버 지도에서 검색", uiJSONAction(uiObject{"open": uiObject{"type": "url", "src": quick(mapSearchURL, q.Word)}}))
	b["enabled"] = q.Word != ""
	return b
}

func (q mapUIQuery) prepare() uiObject {
	message := "검색 중…"
	if q.Word == "" {
		message = "장소, 주소 또는 버스 번호를 입력하세요."
	}
	panel := uiView(uiLabel(message), q.webButton())
	panel["id"] = q.id()
	return uiUpdate("map_content", uiObject{"components": []uiObject{panel}})
}

func (q mapUIQuery) response(results []mapResult, status, placeholder string) uiObject {
	id := q.id()
	listID := id + "_results"
	rows := []uiObject{}
	for i, r := range results {
		rows = append(rows, uiObject{
			"uid": fmt.Sprintf("%s_%d", r.Kind, i), "name.text": r.Name,
			"detail.text": r.Detail, "kind.text": map[string]string{"place": "장소", "address": "주소", "bus": "버스"}[r.Kind],
			"name": r.Name, "address": r.Address, "url": r.URL, "hasaddress": r.Address != "",
		})
	}
	selected := func(field string) string { return "{value:" + listID + ".selected." + field + "}" }
	selectionAction := func(field, command, typ string) uiObject {
		return uiObject{"type": "json", "variables": uiObject{"map_value": selected(field)},
			"src": uiObject{command: uiObject{"type": typ, "src": "{var:map_value}"}}}
	}
	cell := uiView(uiObject{"id": "name", "type": "label"}, uiObject{"id": "detail", "type": "label", "colour": "secondaryLabelColor"}, uiObject{"id": "kind", "type": "label", "colour": "secondaryLabelColor", "font": uiObject{"size": "small"}})
	cell["id"] = "map_cell"
	cell["padding"] = 8
	cell["spacing"] = 2
	list := uiObject{
		"id": listID, "type": "dataview", "placeholder": placeholder,
		"constraints": uiObject{"size": uiObject{"height": 320}},
		"cell":        uiObject{"height": 82, "root": cell, "init": uiObject{"map_cell.background": nil, "name.colour": "labelColor", "detail.colour": "secondaryLabelColor", "kind.colour": "secondaryLabelColor"}, "selected": uiObject{"map_cell.background": "alfredSelectedResultBackgroundColor", "name.colour": "alfredSelectedResultTextColor", "detail.colour": "alfredSelectedResultTextColor", "kind.colour": "alfredSelectedResultTextColor"}},
		"data":        uiObject{"type": "basic", "src": "{var:" + id + "_rows}"},
		"events": []uiObject{
			{"triggers": "actioned", "action": selectionAction("url", "open", "url")},
			{"triggers": "changed", "action": uiObject{"type": "json", "variables": uiObject{"map_has_address": selected("hasaddress")}, "src": uiUpdate(id+"_copy_address", uiObject{"enabled": "{var:map_has_address}"})}},
		},
	}
	open := uiButton("선택한 결과 열기", selectionAction("url", "open", "url"))
	open["enabled"] = len(rows) > 0
	open["shortcut"] = "⌘↩︎"
	name := uiButton("이름 복사", selectionAction("name", "copy", "text"))
	name["enabled"] = len(rows) > 0
	address := uiButton("주소 복사", selectionAction("address", "copy", "text"))
	address["id"] = id + "_copy_address"
	address["enabled"] = len(results) > 0 && results[0].Address != ""
	buttons := uiObject{"type": "view", "layout": "inline", "direction": "horizontal", "spacing": 8, "components": []uiObject{open, name, address}}
	out := uiUpdate(id, uiView(uiLabel(status), list, buttons, q.webButton()))
	// Variables are not preprocessed, and a resolved {var:} is not expanded again.
	// Keep remote titles/addresses containing {file:...} or {var:...} literal.
	out["variables"] = uiObject{id + "_rows": rows}
	return out
}

func (r *request) searchMapUI(q mapUIQuery) uiObject {
	if q.Word == "" {
		return q.response(nil, "검색어를 입력하세요.", "장소, 주소 또는 버스 번호")
	}
	loc, fallback := r.getLocationInfo(q.Mode == "useIP")
	locationText := "설정한 위치 기준"
	if q.Mode == "useIP" {
		locationText = "IP 추정 위치 기준"
	}
	if fallback {
		locationText = "IP 위치 조회 실패 · 설정한 위치를 사용합니다"
	}
	root, err := r.mapDataAt(q.Mode == "useIP", q.Word, loc)
	if err != nil {
		logf("map UI: %v", err)
		return q.response(nil, locationText+" · 검색에 실패했습니다. 다시 검색하거나 네이버 지도를 열어주세요.", "결과를 불러오지 못했습니다")
	}
	results := mapResults(root, q.Category)
	return q.response(results, fmt.Sprintf("%s · %d개 결과", locationText, len(results)), "이 카테고리에 검색 결과가 없습니다")
}

// MapUI emits Alfred UI JSON actions, never Script Filter feedback or update
// notices. The UI scripts use this dedicated protocol even for workflow:update.
func MapUI(command string, args []string) {
	q := newMapUIQuery(args)
	var out uiObject
	if command == "map-ui-prepare" {
		out = q.prepare()
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
		defer cancel()
		r := &request{ctx: ctx, http: httpx.New(&http.Client{})}
		out = r.searchMapUI(q)
	}
	_ = json.NewEncoder(os.Stdout).Encode(out)
}

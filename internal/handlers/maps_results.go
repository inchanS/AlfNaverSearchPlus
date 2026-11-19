package handlers

import (
	"fmt"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/urlx"
)

// Map results are independent of either Alfred presentation format.
type mapResult struct {
	Name, Address, Detail, URL, Kind string
}

func decodeMapResult(kind string, o map[string]any) mapResult {
	r := mapResult{Kind: kind, Name: jsonx.AsStr(o["title"])}
	switch kind {
	case "place":
		r.Address = jsonx.AsStr(o["roadAddress"])
		if r.Address == "" {
			r.Address = jsonx.AsStr(o["jibunAddress"])
		}
		r.Detail = r.Address
		r.URL = fmt.Sprintf(mapSearchTypeURL, urlx.Quote(r.Name), urlx.Quote(jsonx.AsStr(o["type"])), urlx.Quote(jsonx.AsStr(o["id"])))
	case "address":
		r.Name = jsonx.AsStr(o["fullAddress"])
		r.Address = r.Name
		r.Detail = jsonx.AsStr(o["title"])
		r.URL = fmt.Sprintf(mapEntryURL, "address", urlx.Quote(jsonx.AsStr(o["y"])), urlx.Quote(jsonx.AsStr(o["x"])), urlx.Quote(r.Name))
	case "bus":
		r.Detail = jsonx.AsStr(o["cityName"]) + "버스 " + r.Name
		r.URL = fmt.Sprintf(mapSearchTypeURL, urlx.Quote(r.Name), "bus-route", urlx.Quote(jsonx.AsStr(o["id"])))
	}
	return r
}

func mapResults(root map[string]any, category string) []mapResult {
	results := []mapResult{}
	add := func(kind string, o map[string]any) {
		r := decodeMapResult(kind, o)
		if r.Name != "" {
			results = append(results, r)
		}
	}
	if category == "all" {
		for _, el := range jsonx.Arr(root["all"]) {
			o := jsonx.Obj(el)
			for _, kind := range []string{"address", "place", "bus"} {
				if o[kind] != nil {
					add(kind, jsonx.Obj(o[kind]))
					break
				}
			}
		}
	} else {
		for _, el := range jsonx.Arr(root[category]) {
			add(category, jsonx.Obj(el))
		}
	}
	return results
}

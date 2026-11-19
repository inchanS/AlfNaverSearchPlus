package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/env"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

const (
	apiLocation      = "https://map.naver.com/p/api/location"
	apiSearch        = "https://map.naver.com/p/api/search/instant-search"
	mapSearchURL     = "https://map.naver.com/p/search/%s"
	mapEntryURL      = "https://map.naver.com/p/entry/%s/%s,%s,%s"
	mapSearchTypeURL = "https://map.naver.com/p/search/%s/%s/%s"
)

const locationTimeout = time.Second

func mapHeaders() map[string]string {
	return map[string]string{"Referer": "https://map.naver.com/"}
}

type location struct{ lat, lng string }

func decodeLocation(data []byte) (location, error) {
	root, err := jsonx.DecodeObject(data)
	if err != nil {
		return location{}, err
	}
	loc := location{jsonx.AsStr(root["lat"]), jsonx.AsStr(root["lng"])}
	for i, value := range []string{loc.lat, loc.lng} {
		n, err := strconv.ParseFloat(value, 64)
		limit := 90.0
		if i == 1 {
			limit = 180
		}
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > limit {
			return location{}, fmt.Errorf("invalid location coordinates")
		}
	}
	return loc, nil
}

// getLocation reserves time for the search even if IP lookup is unavailable.
func (r *request) getLocation(useIP bool) location {
	loc, _ := r.getLocationInfo(useIP)
	return loc
}

// getLocationInfo also reports when IP lookup fell back to configured coordinates.
func (r *request) getLocationInfo(useIP bool) (location, bool) {
	if useIP {
		loc, err := cache.Load("location_data", env.CacheAge(), func() ([]byte, error) {
			ctx, cancel := context.WithTimeout(r.ctx, locationTimeout)
			defer cancel()
			body, err := r.http.Get(ctx, apiLocation, nil, mapHeaders())
			if err != nil {
				return nil, err
			}
			root, err := jsonx.DecodeObject(body)
			if err != nil {
				return nil, err
			}
			return json.Marshal(root["lngLat"])
		}, decodeLocation)
		if err == nil {
			return loc, false
		}
		logf("map: location lookup failed, using default coordinates: %v", err)
	}
	return location{env.Latitude(), env.Longitude()}, useIP
}

// mapData shares the complete response across overview and category searches.
// The v2 key includes the actual coordinates; legacy query-only caches cannot
// accidentally supply results from another location.
func (r *request) mapData(useIP bool, word string) (map[string]any, error) {
	loc := r.getLocation(useIP)
	return r.mapDataAt(useIP, word, loc)
}

func (r *request) mapDataAt(useIP bool, word string, loc location) (map[string]any, error) {
	key := cache.Key("navmap2", strconv.FormatBool(useIP)+"\x00"+loc.lat+"\x00"+loc.lng+"\x00"+word)
	return r.cachedJSON(key, env.CacheAge(), apiSearch, map[string]string{
		"query": word, "type": "all", "coords": loc.lat + "," + loc.lng,
		"lang": "ko", "caller": "pcweb",
	}, mapHeaders())
}

func addMapSearch(fb *alfred.Feedback, word, prefix string) {
	url := quick(mapSearchURL, word)
	fb.Add(alfred.ItemOpts{
		Title:        prefix + " '" + word + "'",
		Autocomplete: word,
		Arg:          url,
		QuicklookURL: url,
		Valid:        true,
	})
}

func addNoResults(fb *alfred.Feedback, word string) {
	fb.Add(alfred.ItemOpts{
		Title: "No search results for '" + word + "'",
		Icon:  iconNoResults,
		Valid: false,
	})
}

func addMapResult(fb *alfred.Feedback, result mapResult, prefix, icon string) {
	fb.Add(alfred.ItemOpts{
		Title: prefix + " '" + result.Name + "'", Subtitle: result.Detail,
		Autocomplete: result.Name, Arg: result.URL, Copy: result.Name,
		LargeType: result.Name, QuicklookURL: result.URL, Icon: icon, Valid: true,
	})
}

func addPlace(fb *alfred.Feedback, place map[string]any, prefix, icon string) {
	addMapResult(fb, decodeMapResult("place", place), prefix, icon)
}
func addAddress(fb *alfred.Feedback, address map[string]any, prefix, icon string) {
	addMapResult(fb, decodeMapResult("address", address), prefix, icon)
}
func addBus(fb *alfred.Feedback, bus map[string]any, prefix, icon string) {
	addMapResult(fb, decodeMapResult("bus", bus), prefix, icon)
}

// mapSearch: first-step map search (mode is "useIP" or "doNotUseIP").
func (r *request) mapSearch(fb *alfred.Feedback, mode, word string) error {
	useIP := mode == "useIP"

	phrase := "Search Naver Map for"
	if useIP {
		phrase = "Search Naver Map(ip) for"
	}
	addMapSearch(fb, word, phrase)
	root, err := r.mapData(useIP, word)
	if err != nil {
		return err
	}

	place := jsonx.Arr(jsonx.Get(root, "place"))
	address := jsonx.Arr(jsonx.Get(root, "address"))
	bus := jsonx.Arr(jsonx.Get(root, "bus"))

	if len(place) == 0 && len(address) == 0 && len(bus) == 0 {
		addNoResults(fb, word)
	}
	if len(place) > 0 {
		addCategory(fb, useIP, word, "place", iconPlace, "Search only Place for")
	}
	if len(address) > 0 {
		addCategory(fb, useIP, word, "address", iconAddress, "Search only Address for")
	}
	if len(bus) > 0 {
		addCategory(fb, useIP, word, "bus", iconBus, "Search only Bus for")
	}

	for _, el := range jsonx.Arr(jsonx.Get(root, "all")) {
		o := jsonx.Obj(el)
		switch {
		case o["address"] != nil:
			addAddress(fb, jsonx.Obj(o["address"]), "Search Naver Map for", "")
		case o["place"] != nil:
			addPlace(fb, jsonx.Obj(o["place"]), "Search Naver Map for", "")
		case o["bus"] != nil:
			addBus(fb, jsonx.Obj(o["bus"]), "Search Naver Map for", "")
		}
	}
	return nil
}

// addCategory adds an item that switches to a category-only search. The
// map_type variable both routes the info.plist conditional and is read as an
// env var by the maphub handler.
func addCategory(fb *alfred.Feedback, useIP bool, word, categoryType, icon, prefix string) {
	it := fb.Add(alfred.ItemOpts{
		Title:        prefix + " '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		Icon:         icon,
		Valid:        true,
	})
	it.SetVar("map_type", categoryType)
	it.SetVar("useIP", useIP)
}

// mapHub gets both category and location mode from the selected Alfred item.
func (r *request) mapHub(fb *alfred.Feedback, word string) error {
	mapType := env.Str("map_type", "")

	useIP, _ := strconv.ParseBool(env.Str("useIP", "false"))

	// Keep navigation available even when fetching or decoding results fails.
	addMapSearch(fb, word, "Search Naver Map for")
	ret := fb.Add(alfred.ItemOpts{
		Title:        "Return... search naver map for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		Valid:        true,
	})
	ret.SetVar("useIP", useIP)
	ret.SetVar("return", true)

	switch mapType {
	case "place", "address", "bus":
	default:
		return fmt.Errorf("unknown map category: %s", mapType)
	}
	root, err := r.mapData(useIP, word)
	if err != nil {
		return err
	}
	results := jsonx.Arr(root[mapType])

	if len(results) == 0 {
		addNoResults(fb, word)
	}

	for _, el := range results {
		o := jsonx.Obj(el)
		switch mapType {
		case "place":
			addPlace(fb, o, "Search Place for", iconPlace)
		case "address":
			addAddress(fb, o, "Search Address for", iconAddress)
		case "bus":
			addBus(fb, o, "Search Bus for", iconBus)
		}
	}
	return nil
}

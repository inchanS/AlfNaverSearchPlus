package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/env"
	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/urlx"
)

const (
	apiLocation      = "https://map.naver.com/p/api/location"
	apiSearch        = "https://map.naver.com/p/api/search/instant-search"
	mapSearchURL     = "https://map.naver.com/p/search/%s"
	mapEntryURL      = "https://map.naver.com/p/entry/%s/%s,%s,%s"
	mapSearchTypeURL = "https://map.naver.com/p/search/%s/%s/%s"
)

func mapHeaders() map[string]string {
	return map[string]string{
		"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 " +
			"(KHTML, like Gecko) Version/17.6 Safari/605.1.15",
		"Referer": "https://map.naver.com/",
	}
}

// getIPLatLng fetches the caller's IP-based location and returns the stored
// lngLat object (containing lat/lng) as JSON bytes for caching.
func getIPLatLng() ([]byte, error) {
	body, err := httpx.Get(apiLocation, nil, mapHeaders())
	if err != nil {
		return nil, err
	}
	root, err := jsonx.Decode(body)
	if err != nil {
		return nil, err
	}
	return json.Marshal(jsonx.Get(root, "lngLat"))
}

// getLocation returns lat/lng, using the (cached) IP location when useIP is set
// and falling back to the configured default coordinates on any failure.
func getLocation(useIP bool) (lat, lng string) {
	if useIP {
		data, err := cache.Cached("location_data", env.CacheAge(), getIPLatLng)
		if err == nil && data != nil {
			ll, derr := jsonx.Decode(data)
			if derr == nil {
				lat = jsonx.AsStr(jsonx.Get(ll, "lat"))
				lng = jsonx.AsStr(jsonx.Get(ll, "lng"))
				if lat != "" && lng != "" {
					return lat, lng
				}
			}
		}
		logf("map: location lookup failed, using default coordinates: %v", err)
	}
	return env.Latitude(), env.Longitude()
}

// getData performs the instant-search request and returns the raw body.
func getData(lat, lng, word string) ([]byte, error) {
	return httpx.Get(apiSearch, map[string]string{
		"query": word, "type": "all",
		"coords": lat + "," + lng, "lang": "ko", "caller": "pcweb",
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

func addPlace(fb *alfred.Feedback, place map[string]any, prefix, icon string) {
	txt := jsonx.AsStr(place["title"])
	address := jsonx.AsStr(place["roadAddress"])
	if address == "" {
		address = jsonx.AsStr(place["jibunAddress"])
	}
	url := fmt.Sprintf(mapSearchTypeURL, urlx.Quote(txt), jsonx.AsStr(place["type"]), jsonx.AsStr(place["id"]))
	fb.Add(alfred.ItemOpts{
		Title:        prefix + " '" + txt + "'",
		Subtitle:     address,
		Autocomplete: txt,
		Arg:          url,
		Copy:         txt,
		LargeType:    txt,
		QuicklookURL: url,
		Icon:         icon,
		Valid:        true,
	})
}

func addAddress(fb *alfred.Feedback, address map[string]any, prefix, icon string) {
	txt := jsonx.AsStr(address["fullAddress"])
	url := fmt.Sprintf(mapEntryURL, "address", jsonx.AsStr(address["y"]), jsonx.AsStr(address["x"]), urlx.Quote(txt))
	fb.Add(alfred.ItemOpts{
		Title:        prefix + " '" + txt + "'",
		Subtitle:     jsonx.AsStr(address["title"]),
		Autocomplete: txt,
		Arg:          url,
		Copy:         txt,
		LargeType:    txt,
		QuicklookURL: url,
		Icon:         icon,
		Valid:        true,
	})
}

func addBus(fb *alfred.Feedback, bus map[string]any, prefix, icon string) {
	txt := jsonx.AsStr(bus["title"])
	subtitle := jsonx.AsStr(bus["cityName"]) + "버스 " + txt
	url := fmt.Sprintf(mapSearchTypeURL, urlx.Quote(txt), "bus-route", jsonx.AsStr(bus["id"]))
	fb.Add(alfred.ItemOpts{
		Title:        prefix + " '" + txt + "'",
		Subtitle:     subtitle,
		Autocomplete: txt,
		Arg:          url,
		Copy:         txt,
		LargeType:    txt,
		QuicklookURL: url,
		Icon:         icon,
		Valid:        true,
	})
}

// mapSearch: first-step map search (mode is "useIP" or "doNotUseIP").
func mapSearch(fb *alfred.Feedback, mode, word string) error {
	useIP := mode == "useIP"

	// Persist the IP-usage choice for the follow-up category step.
	if data, err := json.Marshal(map[string]bool{"use": useIP}); err == nil {
		_ = cache.Write("use_ip", data)
	}

	lat, lng := getLocation(useIP)
	phrase, keyPrefix := "Search Naver Map for", "navmap"
	if useIP {
		phrase, keyPrefix = "Search Naver Map(ip) for", "navmapip"
	}

	body, err := cache.Cached(cache.Key(keyPrefix, word), env.CacheAge(), func() ([]byte, error) {
		return getData(lat, lng, word)
	})
	if err != nil {
		return err
	}
	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}

	addMapSearch(fb, word, phrase)

	place := jsonx.Arr(jsonx.Get(root, "place"))
	address := jsonx.Arr(jsonx.Get(root, "address"))
	bus := jsonx.Arr(jsonx.Get(root, "bus"))

	if len(place) == 0 && len(address) == 0 && len(bus) == 0 {
		addNoResults(fb, word)
	}
	if len(place) > 0 {
		addCategory(fb, word, "place", iconPlace, "Search only Place for")
	}
	if len(address) > 0 {
		addCategory(fb, word, "address", iconAddress, "Search only Address for")
	}
	if len(bus) > 0 {
		addCategory(fb, word, "bus", iconBus, "Search only Bus for")
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
func addCategory(fb *alfred.Feedback, word, categoryType, icon, prefix string) {
	it := fb.Add(alfred.ItemOpts{
		Title:        prefix + " '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		Icon:         icon,
		Valid:        true,
	})
	it.SetVar("map_type", categoryType)
}

var hubCacheName = map[string]string{
	"place":   "navplace",
	"address": "navaddress",
	"bus":     "navbus",
}

// mapHub: second-step, category-filtered map search. The category comes from
// the map_type workflow variable (exposed as an env var); IP usage is read from
// the state saved by mapSearch.
func mapHub(fb *alfred.Feedback, word string) error {
	mapType := env.Str("map_type", "")

	useIP := false
	if data, ok := cache.Read("use_ip", 0); ok {
		var s struct {
			Use bool `json:"use"`
		}
		if json.Unmarshal(data, &s) == nil {
			useIP = s.Use
		}
	}

	name := hubCacheName[mapType]
	if name == "" {
		name = "nav" + mapType
	}
	if useIP {
		name += "ip"
	}

	body, err := cache.Cached(cache.Key(name, word), env.CacheAge(), func() ([]byte, error) {
		lat, lng := getLocation(useIP)
		full, gerr := getData(lat, lng, word)
		if gerr != nil {
			return nil, gerr
		}
		root, derr := jsonx.Decode(full)
		if derr != nil {
			return nil, derr
		}
		return json.Marshal(jsonx.Get(root, mapType))
	})
	if err != nil {
		return err
	}

	var cat any
	if body != nil {
		cat, _ = jsonx.Decode(body)
	}
	results := jsonx.Arr(cat)

	if len(results) == 0 {
		addNoResults(fb, word)
	}

	addMapSearch(fb, word, "Search Naver Map for")

	ret := fb.Add(alfred.ItemOpts{
		Title:        "Return... search naver map for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		Valid:        true,
	})
	ret.SetVar("useIP", useIP)
	ret.SetVar("return", true)

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

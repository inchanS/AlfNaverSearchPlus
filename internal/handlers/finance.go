package handlers

import (
	"net/url"
	"strings"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/env"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

// finance: Naver Finance ticker autocomplete.
func (r *request) finance(fb *alfred.Feedback, word string) error {
	root, err := r.cachedJSON(cache.Key("navfinance", word), env.CacheAge(), "https://ac.stock.naver.com/ac", map[string]string{
		"q": word, "target": "index,stock,marketindicator",
		"lang": "ko", "caller": "pcweb",
	}, map[string]string{
		"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 " +
			"(KHTML, like Gecko) Version/15.4 Safari/605.1.15",
	})
	if err != nil {
		return err
	}

	items := jsonx.Arr(jsonx.Get(root, "items"))
	if len(items) == 0 {
		fb.Add(alfred.ItemOpts{
			Title: "No search results for '" + word + "'",
			Icon:  iconNoResults,
			Valid: false,
		})
		return nil
	}

	for _, el := range items {
		code := jsonx.AsStr(jsonx.Get(el, "code"))
		name := jsonx.AsStr(jsonx.Get(el, "name"))
		typeCode := jsonx.AsStr(jsonx.Get(el, "typeCode"))
		nationName := jsonx.AsStr(jsonx.Get(el, "nationName"))

		subtitle := name + ", " + code + ", " + typeCode + ", " + nationName
		itemURL := financeItemURL(jsonx.Str(jsonx.Get(el, "url")))

		fb.Add(alfred.ItemOpts{
			Title:        "Search Naver Finance for '" + name + "'",
			Subtitle:     subtitle,
			Autocomplete: name,
			Arg:          itemURL,
			Copy:         subtitle,
			LargeType:    subtitle,
			QuicklookURL: itemURL,
			Valid:        itemURL != "",
		})
	}
	return nil
}

// financeItemURL uses the API's route, which includes the full exchange-specific
// identifier (e.g. NVDA.O rather than NVDA). Only stock detail routes switch to
// the price tab; indices and market indicators keep their original routes.
// Missing or malformed routes leave the result visible but not actionable.
func financeItemURL(route string) string {
	u, err := url.Parse(route)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil ||
		!strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") || u.Path == "/" {
		return ""
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) >= 2 && (parts[0] == "domestic" || parts[0] == "worldstock") && parts[1] == "stock" {
		if len(parts) != 4 || parts[2] == "" || parts[3] == "" {
			return ""
		}
		if parts[3] == "total" {
			u.Path = strings.TrimSuffix(u.Path, "/total") + "/price"
			u.RawPath = ""
		}
	}
	u.Scheme = "https"
	u.Host = "stock.naver.com"
	return u.String()
}

package handlers

import (
	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/env"
	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

// finance: Naver Finance ticker autocomplete.
func finance(fb *alfred.Feedback, word string) error {
	const itemURL = "https://finance.naver.com/item/main.naver?code=%s"

	body, err := cache.Cached(cache.Key("navfinance", word), env.CacheAge(), func() ([]byte, error) {
		return httpx.Get("https://ac.stock.naver.com/ac", map[string]string{
			"q": word, "target": "index,stock,marketindicator",
			"lang": "ko", "caller": "pcweb",
		}, map[string]string{
			"user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_4) AppleWebKit/605.1.15 " +
				"(KHTML, like Gecko) Version/15.4 Safari/605.1.15",
		})
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
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
		url := quick(itemURL, code)

		fb.Add(alfred.ItemOpts{
			Title:        "Search Naver Finance for '" + name + "'",
			Subtitle:     subtitle,
			Autocomplete: name,
			Arg:          url,
			Copy:         subtitle,
			LargeType:    subtitle,
			QuicklookURL: url,
			Valid:        true,
		})
	}
	return nil
}

package handlers

import (
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

// search: Naver integrated search autocomplete.
func search(fb *alfred.Feedback, word string) error {
	const quicklook = "https://search.naver.com/search.naver?ie=utf8&sm=stp_hty&where=se&query=%s"

	fb.Add(alfred.ItemOpts{
		Title:        "Search Naver for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})

	body, err := cache.Cached(cache.Key("nav", word), 30*time.Second, func() ([]byte, error) {
		return httpx.Get("https://ac.search.naver.com/nx/ac", map[string]string{
			"q_enc": "UTF-8", "st": "100", "r_format": "json", "r_enc": "UTF-8",
			"r_unicode": "0", "t_koreng": "1", "ans": "1", "run": "2", "rev": "4", "q": word,
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, el := range jsonx.Arr(jsonx.At(jsonx.Get(root, "items"), 0)) {
		txt := jsonx.Str(jsonx.At(el, 0))
		if txt == "" {
			continue
		}
		fb.Add(alfred.ItemOpts{
			Title:        "Search Naver for '" + txt + "'",
			Autocomplete: txt,
			Arg:          txt,
			Copy:         txt,
			LargeType:    txt,
			QuicklookURL: quick(quicklook, txt),
			Valid:        true,
		})
	}
	return nil
}

// shopping: Naver Shopping keyword autocomplete.
func shopping(fb *alfred.Feedback, word string) error {
	const quicklook = "https://search.shopping.naver.com/ns/search?query=%s"

	fb.Add(alfred.ItemOpts{
		Title:        "Search Naver Shopping for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})

	body, err := cache.Cached(cache.Key("navs", word), 30*time.Second, func() ([]byte, error) {
		return httpx.Get("https://m.shopping.naver.com/api/modules/gnb/auto-complete", map[string]string{
			"keyword": word, "personalizeYn": "n",
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, el := range jsonx.Arr(jsonx.Get(jsonx.Get(root, "result"), "keywordList")) {
		txt := jsonx.Str(jsonx.Get(el, "keywordName"))
		if txt == "" {
			continue
		}
		fb.Add(alfred.ItemOpts{
			Title:        "Search Naver Shopping for '" + txt + "'",
			Autocomplete: txt,
			Arg:          txt,
			Copy:         txt,
			LargeType:    txt,
			QuicklookURL: quick(quicklook, txt),
			Valid:        true,
		})
	}
	return nil
}

// terms: Naver encyclopedia (terms) autocomplete.
func terms(fb *alfred.Feedback, word string) error {
	const quicklook = "https://terms.naver.com/search.naver?query=%s"

	fb.Add(alfred.ItemOpts{
		Title:        "Search Naver Terms for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})

	body, err := cache.Cached(cache.Key("navterm", word), 30*time.Second, func() ([]byte, error) {
		return httpx.Get("https://ac.terms.naver.com/ac", map[string]string{
			"frm": "terms", "q_enc": "UTF-8", "st": "111111", "r_lt": "111111",
			"r_format": "json", "r_enc": "UTF-8", "r_unicode": "0", "q": word,
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, el := range jsonx.Arr(jsonx.At(jsonx.Get(root, "items"), 0)) {
		txt := jsonx.Str(jsonx.At(el, 0))
		if txt == "" {
			continue
		}
		fb.Add(alfred.ItemOpts{
			Title:        txt,
			Autocomplete: txt,
			Arg:          txt,
			Copy:         txt,
			LargeType:    txt,
			QuicklookURL: quick(quicklook, txt),
			Valid:        true,
		})
	}
	return nil
}

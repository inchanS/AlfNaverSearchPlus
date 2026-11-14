package handlers

import (
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

// search: Naver integrated search autocomplete.
func (r *request) search(fb *alfred.Feedback, word string) error {
	const quicklook = "https://search.naver.com/search.naver?ie=utf8&sm=stp_hty&where=se&query=%s"

	addQuery(fb, word, "Search Naver for '"+word+"'", quicklook)

	root, err := r.cachedJSON(cache.Key("nav", word), 30*time.Second, "https://ac.search.naver.com/nx/ac", map[string]string{
		"q_enc": "UTF-8", "st": "100", "r_format": "json", "r_enc": "UTF-8",
		"r_unicode": "0", "t_koreng": "1", "ans": "1", "run": "2", "rev": "4", "q": word,
	}, nil)
	if err != nil {
		return err
	}

	for _, el := range jsonx.Arr(jsonx.At(jsonx.Get(root, "items"), 0)) {
		txt := jsonx.Str(jsonx.At(el, 0))
		if txt == "" {
			continue
		}
		addSuggestion(fb, txt, "Search Naver for '"+txt+"'", "", txt, quicklook)
	}
	return nil
}

// shopping: Naver Shopping keyword autocomplete.
func (r *request) shopping(fb *alfred.Feedback, word string) error {
	const quicklook = "https://search.shopping.naver.com/ns/search?query=%s"

	addQuery(fb, word, "Search Naver Shopping for '"+word+"'", quicklook)

	root, err := r.cachedJSON(cache.Key("navs", word), 30*time.Second, "https://m.shopping.naver.com/api/modules/gnb/auto-complete", map[string]string{
		"keyword": word, "personalizeYn": "n",
	}, nil)
	if err != nil {
		return err
	}

	for _, el := range jsonx.Arr(jsonx.Get(jsonx.Get(root, "result"), "keywordList")) {
		txt := jsonx.Str(jsonx.Get(el, "keywordName"))
		if txt == "" {
			continue
		}
		addSuggestion(fb, txt, "Search Naver Shopping for '"+txt+"'", "", txt, quicklook)
	}
	return nil
}

// terms: Naver encyclopedia (terms) autocomplete.
func (r *request) terms(fb *alfred.Feedback, word string) error {
	const quicklook = "https://terms.naver.com/search.naver?query=%s"

	addQuery(fb, word, "Search Naver Terms for '"+word+"'", quicklook)

	root, err := r.cachedJSON(cache.Key("navterm", word), 30*time.Second, "https://ac.terms.naver.com/ac", map[string]string{
		"frm": "terms", "q_enc": "UTF-8", "st": "111111", "r_lt": "111111",
		"r_format": "json", "r_enc": "UTF-8", "r_unicode": "0", "q": word,
	}, nil)
	if err != nil {
		return err
	}

	for _, el := range jsonx.Arr(jsonx.At(jsonx.Get(root, "items"), 0)) {
		txt := jsonx.Str(jsonx.At(el, 0))
		if txt == "" {
			continue
		}
		addSuggestion(fb, txt, txt, "", txt, quicklook)
	}
	return nil
}

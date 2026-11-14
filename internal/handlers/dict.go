package handlers

import (
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

const dictCacheAge = 600 * time.Second

// dictMulti: multilingual dictionary (e.g. jako, deko …). lang is the API path
// segment; its first two letters name the dictionary in the UI.
func (r *request) dictMulti(fb *alfred.Feedback, lang, word string) error {
	quicklook := "https://dict.naver.com/" + lang + "dict/#/search?query=%s"
	label := "Search Naver " + lang[:min(2, len(lang))] + "dic for"

	head := addQuery(fb, word, label+" '"+word+"'", quicklook)
	head.SetVar("lang", lang)

	root, err := r.cachedJSON(cache.Key(lang, word), dictCacheAge, "https://ac-dict.naver.com/"+lang+"/ac", map[string]string{
		"q": word, "_callback": "", "q_enc": "UTF-8", "st": "11",
		"r_lt": "10", "r_format": "json", "r_enc": "UTF-8",
	}, nil)
	if err != nil {
		return err
	}

	for _, item := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, ltxt := range jsonx.Arr(item) {
			txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
			if txt == "" {
				continue
			}
			rtxt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 3), 0))
			it := addSuggestion(fb, txt, txt+"     "+rtxt, label+" '"+txt+"'", rtxt, quicklook)
			it.SetVar("lang", lang)
		}
	}
	return nil
}

// endic: English → Korean dictionary.
func (r *request) endic(fb *alfred.Feedback, word string) error {
	const quicklook = "https://en.dict.naver.com/#/search?query=%s"

	head := addQuery(fb, word, "Search Naver Endic for '"+word+"'", quicklook)
	head.SetVar("lang", "enko")

	root, err := r.cachedJSON(cache.Key("en", word), dictCacheAge, "https://ac-dict.naver.com/enko/ac", map[string]string{
		"q_enc": "utf-8", "st": "11001", "r_format": "json", "r_enc": "utf-8",
		"r_lt": "10001", "r_unicode": "0", "r_escape": "1", "q": word,
	}, nil)
	if err != nil {
		return err
	}

	for _, items := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, item := range jsonx.Arr(items) {
			txt := jsonx.Str(jsonx.At(jsonx.At(item, 0), 0))
			if txt == "" {
				continue
			}
			rtxt := jsonx.Str(jsonx.At(jsonx.At(item, 2), 0))
			it := addSuggestion(fb, txt, txt+"     "+rtxt, "Search Naver Endic for '"+txt+"'", rtxt, quicklook)
			it.SetVar("lang", "enko")
		}
	}
	return nil
}

// krdic: Korean dictionary.
func (r *request) krdic(fb *alfred.Feedback, word string) error {
	const quicklook = "https://ko.dict.naver.com/#/search?query=%s"

	head := addQuery(fb, word, "Search Naver Krdic for '"+word+"'", quicklook)
	head.SetVar("lang", "koko")

	root, err := r.cachedJSON(cache.Key("kr", word), dictCacheAge, "https://ac-dict.naver.com/koko/ac", map[string]string{
		"frm": "stdkrdic", "oe": "utf8", "m": "0", "r": "1",
		"st": "111", "r_lt": "111", "q": word,
	}, nil)
	if err != nil {
		return err
	}

	for _, items := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, ltxt := range jsonx.Arr(items) {
			txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
			if txt == "" {
				continue
			}
			it := addSuggestion(fb, txt, txt, "Search Naver Krdic for '"+txt+"'", txt, quicklook)
			it.SetVar("lang", "koko")
		}
	}
	return nil
}

// hanja: Hanja (Chinese-character) dictionary.
func (r *request) hanja(fb *alfred.Feedback, word string) error {
	const quicklook = "https://hanja.dict.naver.com/#/search?range=all&query=%s"

	addQuery(fb, word, "Search Naver Hanja for '"+word+"'", quicklook)

	root, err := r.cachedJSON(cache.Key("hanja", word), dictCacheAge, "https://ac-dict.naver.com/ccko/ac", map[string]string{
		"st": "11", "r_lt": "11", "q": word,
	}, nil)
	if err != nil {
		return err
	}

	for _, item := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, ltxt := range jsonx.Arr(item) {
			txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
			if txt == "" {
				continue
			}
			rtxt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 1), 0))
			r2txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 3), 0))
			addSuggestion(fb, txt, txt+"["+rtxt+"] "+r2txt, "Search Naver Hanja for '"+txt+"'", r2txt, quicklook)
		}
	}
	return nil
}

// enendic: English → English dictionary.
func (r *request) enendic(fb *alfred.Feedback, word string) error {
	const quicklook = "https://dict.naver.com/enendict/#/search?query=%s"

	head := addQuery(fb, word, "Search Naver Endic for '"+word+"'", quicklook)
	head.SetVar("lang", "enen")

	root, err := r.cachedJSON(cache.Key("en1", word), dictCacheAge, "https://ac-dict.naver.com/enen3/ac", map[string]string{
		"st": "11", "r_format": "json", "r_lt": "11", "q": word,
	}, nil)
	if err != nil {
		return err
	}

	for _, ltxt := range jsonx.Arr(jsonx.At(jsonx.Get(root, "items"), 0)) {
		txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
		if txt == "" {
			continue
		}
		rtxt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 3), 0))
		it := addSuggestion(fb, txt, txt+"     "+rtxt, "Search Naver Endic for '"+txt+"'", rtxt, quicklook)
		it.SetVar("lang", "enen")
	}
	return nil
}

package handlers

import (
	"html"
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
)

const dictCacheAge = 600 * time.Second

// dictMulti: multilingual dictionary (e.g. jako, deko …). lang is the API path
// segment; its first two letters name the dictionary in the UI.
func dictMulti(fb *alfred.Feedback, lang, word string) error {
	quicklook := "https://dict.naver.com/" + lang + "dict/#/search?query=%s"
	label := "Search Naver " + lang[:min(2, len(lang))] + "dic for"

	head := fb.Add(alfred.ItemOpts{
		Title:        label + " '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})
	head.SetVar("lang", lang)

	body, err := cache.Cached(cache.Key(lang, word), dictCacheAge, func() ([]byte, error) {
		return httpx.Get("https://ac-dict.naver.com/"+lang+"/ac", map[string]string{
			"q": word, "_callback": "", "q_enc": "UTF-8", "st": "11",
			"r_lt": "10", "r_format": "json", "r_enc": "UTF-8",
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, item := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, ltxt := range jsonx.Arr(item) {
			txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
			if txt == "" {
				continue
			}
			rtxt := html.EscapeString(jsonx.Str(jsonx.At(jsonx.At(ltxt, 3), 0)))
			it := fb.Add(alfred.ItemOpts{
				Title:        txt + "     " + rtxt,
				Subtitle:     label + " '" + txt + "'",
				Autocomplete: txt,
				Arg:          txt,
				Copy:         rtxt,
				LargeType:    txt,
				QuicklookURL: quick(quicklook, txt),
				Valid:        true,
			})
			it.SetVar("lang", lang)
		}
	}
	return nil
}

// endic: English → Korean dictionary.
func endic(fb *alfred.Feedback, word string) error {
	const quicklook = "https://en.dict.naver.com/#/search?query=%s"

	head := fb.Add(alfred.ItemOpts{
		Title:        "Search Naver Endic for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})
	head.SetVar("lang", "enko")

	body, err := cache.Cached(cache.Key("en", word), dictCacheAge, func() ([]byte, error) {
		return httpx.Get("https://ac-dict.naver.com/enko/ac", map[string]string{
			"q_enc": "utf-8", "st": "11001", "r_format": "json", "r_enc": "utf-8",
			"r_lt": "10001", "r_unicode": "0", "r_escape": "1", "q": word,
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
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
			it := fb.Add(alfred.ItemOpts{
				Title:        txt + "     " + rtxt,
				Subtitle:     "Search Naver Endic for '" + txt + "'",
				Autocomplete: txt,
				Arg:          txt,
				Copy:         rtxt,
				LargeType:    txt,
				QuicklookURL: quick(quicklook, txt),
				Valid:        true,
			})
			it.SetVar("lang", "enko")
		}
	}
	return nil
}

// krdic: Korean dictionary.
func krdic(fb *alfred.Feedback, word string) error {
	const quicklook = "https://ko.dict.naver.com/#/search?query=%s"

	head := fb.Add(alfred.ItemOpts{
		Title:        "Search Naver Krdic for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})
	head.SetVar("lang", "koko")

	body, err := cache.Cached(cache.Key("kr", word), dictCacheAge, func() ([]byte, error) {
		return httpx.Get("https://ac-dict.naver.com/koko/ac", map[string]string{
			"frm": "stdkrdic", "oe": "utf8", "m": "0", "r": "1",
			"st": "111", "r_lt": "111", "q": word,
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, items := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, ltxt := range jsonx.Arr(items) {
			txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
			if txt == "" {
				continue
			}
			it := fb.Add(alfred.ItemOpts{
				Title:        txt,
				Subtitle:     "Search Naver Krdic for '" + txt + "'",
				Autocomplete: txt,
				Arg:          txt,
				Copy:         txt,
				LargeType:    txt,
				QuicklookURL: quick(quicklook, txt),
				Valid:        true,
			})
			it.SetVar("lang", "koko")
		}
	}
	return nil
}

// hanja: Hanja (Chinese-character) dictionary.
func hanja(fb *alfred.Feedback, word string) error {
	const quicklook = "https://hanja.dict.naver.com/#/search?range=all&query=%s"

	fb.Add(alfred.ItemOpts{
		Title:        "Search Naver Hanja for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})

	body, err := cache.Cached(cache.Key("hanja", word), dictCacheAge, func() ([]byte, error) {
		return httpx.Get("https://ac-dict.naver.com/ccko/ac", map[string]string{
			"st": "11", "r_lt": "11", "q": word,
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, item := range jsonx.Arr(jsonx.Get(root, "items")) {
		for _, ltxt := range jsonx.Arr(item) {
			txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
			if txt == "" {
				continue
			}
			rtxt := html.EscapeString(jsonx.Str(jsonx.At(jsonx.At(ltxt, 1), 0)))
			r2txt := html.EscapeString(jsonx.Str(jsonx.At(jsonx.At(ltxt, 3), 0)))
			fb.Add(alfred.ItemOpts{
				Title:        txt + "[" + rtxt + "] " + r2txt,
				Subtitle:     "Search Naver Hanja for '" + txt + "'",
				Autocomplete: txt,
				Arg:          txt,
				Copy:         r2txt,
				LargeType:    txt,
				QuicklookURL: quick(quicklook, txt),
				Valid:        true,
			})
		}
	}
	return nil
}

// enendic: English → English dictionary.
func enendic(fb *alfred.Feedback, word string) error {
	const quicklook = "https://dict.naver.com/enendict/#/search?query=%s"

	head := fb.Add(alfred.ItemOpts{
		Title:        "Search Naver Endic for '" + word + "'",
		Autocomplete: word,
		Arg:          word,
		QuicklookURL: quick(quicklook, word),
		Valid:        true,
	})
	head.SetVar("lang", "enen")

	body, err := cache.Cached(cache.Key("en1", word), dictCacheAge, func() ([]byte, error) {
		return httpx.Get("https://ac-dict.naver.com/enen3/ac", map[string]string{
			"st": "11", "r_format": "json", "r_lt": "11", "q": word,
		}, nil)
	})
	if err != nil {
		return err
	}

	root, err := jsonx.Decode(body)
	if err != nil {
		return err
	}
	for _, ltxt := range jsonx.Arr(jsonx.At(jsonx.Get(root, "items"), 0)) {
		txt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 0), 0))
		if txt == "" {
			continue
		}
		rtxt := jsonx.Str(jsonx.At(jsonx.At(ltxt, 1), 0))
		it := fb.Add(alfred.ItemOpts{
			Title:        txt + "     " + rtxt,
			Subtitle:     "Search Naver Endic for '" + txt + "'",
			Autocomplete: txt,
			Arg:          txt,
			Copy:         rtxt,
			LargeType:    txt,
			QuicklookURL: quick(quicklook, txt),
			Valid:        true,
		})
		it.SetVar("lang", "enen")
	}
	return nil
}

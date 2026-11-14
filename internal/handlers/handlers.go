// Package handlers implements each Alfred entry point (search, dictionaries,
// shopping, finance, maps) and dispatches to them by subcommand name.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
	"github.com/inchanS/AlfNaverSearchPlus/internal/httpx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/jsonx"
	"github.com/inchanS/AlfNaverSearchPlus/internal/update"
	"github.com/inchanS/AlfNaverSearchPlus/internal/urlx"
)

// Icon file names bundled in the workflow directory.
const (
	iconNoResults = "noresults.png"
	iconPlace     = "7FBDB33A-E342-411C-B00B-8B797AE8C19A.png"
	iconAddress   = "3F6E3BB6-64CC-481E-990D-F3823D3616A8.png"
	iconBus       = "845B46E7-61FB-43CD-A287-FCB4C075A4A6.png"
)

const searchTimeout = 3 * time.Second

// request shares one deadline across every network operation in a search.
type request struct {
	ctx  context.Context
	http *httpx.Client
}

func (r *request) cachedJSON(key string, age time.Duration, endpoint string, params, headers map[string]string) (map[string]any, error) {
	return cache.Load(key, age, func() ([]byte, error) {
		return r.http.Get(r.ctx, endpoint, params, headers)
	}, jsonx.DecodeObject)
}

// addQuery supplies the fields shared by fallback and suggestion rows.
func addQuery(fb *alfred.Feedback, word, title, quicklook string) *alfred.Item {
	return fb.Add(alfred.ItemOpts{
		Title: title, Autocomplete: word, Arg: word,
		QuicklookURL: quick(quicklook, word), Valid: true,
	})
}

func addSuggestion(fb *alfred.Feedback, word, title, subtitle, copyText, quicklook string) *alfred.Item {
	it := addQuery(fb, word, title, quicklook)
	it.Subtitle = subtitle
	it.Text = &alfred.Text{Copy: copyText, LargeType: word}
	return it
}

// Dispatch runs the handler named cmd with the remaining CLI args.
func Dispatch(cmd string, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	defer cancel()
	r := &request{ctx: ctx, http: httpx.New(&http.Client{})}
	r.dispatch(cmd, args)
}

func (r *request) dispatch(cmd string, args []string) {
	switch cmd {
	case "search":
		run(func(fb *alfred.Feedback) error { return r.search(fb, at(args, 0)) })
	case "shopping":
		run(func(fb *alfred.Feedback) error { return r.shopping(fb, at(args, 0)) })
	case "terms":
		run(func(fb *alfred.Feedback) error { return r.terms(fb, at(args, 0)) })
	case "dict":
		run(func(fb *alfred.Feedback) error { return r.dictMulti(fb, at(args, 0), at(args, 1)) })
	case "endic":
		run(func(fb *alfred.Feedback) error { return r.endic(fb, at(args, 0)) })
	case "krdic":
		run(func(fb *alfred.Feedback) error { return r.krdic(fb, at(args, 0)) })
	case "hanja":
		run(func(fb *alfred.Feedback) error { return r.hanja(fb, at(args, 0)) })
	case "enendic":
		run(func(fb *alfred.Feedback) error { return r.enendic(fb, at(args, 0)) })
	case "finance":
		run(func(fb *alfred.Feedback) error { return r.finance(fb, at(args, 0)) })
	case "map":
		run(func(fb *alfred.Feedback) error { return r.mapSearch(fb, at(args, 0), at(args, 1)) })
	case "maphub":
		run(func(fb *alfred.Feedback) error { return r.mapHub(fb, at(args, 0)) })
	default:
		fb := alfred.New()
		fb.Add(alfred.ItemOpts{Title: "Unknown command: " + cmd, Valid: false})
		fb.Send()
	}
}

// run wraps a handler body with the update notice, panic/error recovery, and
// final feedback emission. On error it appends an error row while KEEPING any
// items already added — most handlers add a "Search Naver for 'word'" fallback
// row before the network fetch, so a transient/offline failure must not wipe it:
// the user can still press Enter to run the web search. When nothing was added,
// the error row is all that remains, so the user still gets feedback.
func run(body func(fb *alfred.Feedback) error) {
	fb := alfred.New()
	update.MaybeShow(fb)
	if err := safe(fb, body); err != nil {
		fb.Add(alfred.ItemOpts{
			Title:    "Error",
			Subtitle: err.Error(),
			Icon:     iconNoResults,
			Valid:    false,
		})
	}
	fb.Send()
}

// safe invokes body, converting a panic into an error.
func safe(fb *alfred.Feedback, body func(*alfred.Feedback) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return body(fb)
}

// at returns args[i] or "" when out of range.
func at(args []string, i int) string {
	if i >= 0 && i < len(args) {
		return args[i]
	}
	return ""
}

// quick renders a query into a URL template of the form "…query=%s".
func quick(tmpl, word string) string {
	return fmt.Sprintf(tmpl, urlx.Quote(word))
}

// logf writes to stderr, never stdout (which is reserved for Alfred feedback).
func logf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

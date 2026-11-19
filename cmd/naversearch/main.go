// Command naversearch is the single universal binary backing every Alfred
// script filter in the NaverSearchPlus workflow. It dispatches by subcommand:
//
//	naversearch <handler> [args...]   run a search handler (search, dict, map …)
//	naversearch _update-check         background: refresh update availability
//
// When the final argument is the update magic ("workflow:update"), it downloads
// and installs the latest release instead of running a handler.
package main

import (
	"encoding/json"
	"os"

	"golang.org/x/text/unicode/norm"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/handlers"
	"github.com/inchanS/AlfNaverSearchPlus/internal/update"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		return
	}

	normalizeArgs(args)

	cmd := args[0]

	if cmd == "map-ui-launch" {
		mode := "doNotUseIP"
		if len(args) > 1 && args[1] == "useIP" {
			mode = "useIP"
		}
		word := ""
		if len(args) > 2 {
			word = args[2]
		}
		// Preserve the existing background update check and cache cleanup for
		// people who only use maps, without mixing Script Filter JSON into UI.
		notice := alfred.New()
		update.MaybeShow(notice)
		title := "네이버 지도"
		if len(notice.Items) > 0 {
			title += " · 새 버전 있음: n workflow:update"
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"alfredworkflow": map[string]any{"arg": []string{word, mode, title}}})
		return
	}

	// Alfred UI has its own output protocol; even a literal workflow:update
	// in its search field is a query, never an installation command.
	if cmd == "map-ui-prepare" || cmd == "map-ui-search" {
		handlers.MapUI(cmd, args[1:])
		return
	}

	// Detached background worker: refresh update info, no Alfred output.
	if cmd == "_update-check" {
		update.RunCheck()
		return
	}

	// Update magic: the query autocompletes to update.Magic, so any handler
	// invocation ending in it means "install the update".
	if args[len(args)-1] == update.Magic {
		update.Install(alfred.New())
		return
	}

	handlers.Dispatch(cmd, args[1:])
}

// normalizeArgs rewrites each argument to Unicode NFC in place. macOS delivers
// argv in NFD (decomposed) form, but Naver's APIs expect NFC (composed) — e.g.
// the Hangul "스타" must be composed syllables, not separate jamo. This
// reproduces what the old Python library did and also fixes Japanese dakuten,
// German umlauts, etc. ASCII arguments (handler names, "useIP", the update
// magic) are unaffected.
func normalizeArgs(args []string) {
	for i, a := range args {
		args[i] = norm.NFC.String(a)
	}
}

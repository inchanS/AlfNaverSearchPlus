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
	"os"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/handlers"
	"github.com/inchanS/AlfNaverSearchPlus/internal/update"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		return
	}
	cmd := args[0]

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

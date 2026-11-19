package workflow_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapUIEntrypointsAndGraph(t *testing.T) {
	root := parsePlist(t, readFile(t, "../../workflow/info.plist"))
	objects := map[string]plistNode{}
	for _, o := range root.value("objects").Children {
		objects[o.value("uid").Text] = o
	}
	connections := root.value("connections")
	next := func(id string) plistNode {
		t.Helper()
		edges := connections.value(id).Children
		if len(edges) != 1 {
			t.Fatalf("%s: expected one connection, got %d", id, len(edges))
		}
		return objects[edges[0].value("destinationuid").Text]
	}
	modes := map[string]string{"nm": "doNotUseIP", "nmi": "useIP"}
	count := 0
	for _, o := range objects {
		kind := o.value("type").Text
		key := o.value("config").value("keyword").Text
		if kind == "alfred.workflow.trigger.external" {
			key = o.value("config").value("triggerid").Text
		}
		mode, ok := modes[key]
		if !ok {
			continue
		}
		count++
		if kind != "alfred.workflow.input.keyword" && kind != "alfred.workflow.trigger.external" {
			t.Fatal("map entry is not a keyword/trigger")
		}
		launch := next(o.value("uid").Text)
		if !strings.Contains(launch.value("config").value("script").Text, "map-ui-launch "+mode) {
			t.Fatal("wrong location mode")
		}
		ui := next(launch.value("uid").Text)
		if ui.value("type").Text != "alfred.workflow.userinterface.alfredui" || ui.value("config").value("path").Text != "map.alfredui" {
			t.Fatal("map does not reach UI")
		}
		if ui.value("config").value("launchmode").Text != "0" {
			t.Fatal("map should open inside Alfred")
		}
	}
	if count != 4 {
		t.Fatalf("expected two keywords and two external triggers, got %d", count)
	}
	for i := 0; i+1 < len(connections.Children); i += 2 {
		if _, ok := objects[connections.Children[i].Text]; !ok {
			t.Fatal("dangling source")
		}
		for _, edge := range connections.Children[i+1].Children {
			if _, ok := objects[edge.value("destinationuid").Text]; !ok {
				t.Fatal("dangling destination")
			}
		}
	}
}

func TestMapUIAsyncSearchAndConfiguration(t *testing.T) {
	var root map[string]any
	if err := json.Unmarshal(readFile(t, "../../workflow/map.alfredui/ui.json"), &root); err != nil {
		t.Fatal(err)
	}
	config := root["variablesconfig"].(map[string]any)
	for _, key := range []string{"latitude", "longitude", "cache_age"} {
		c := config[key].(map[string]any)
		if c["src"] != "variable" || c["scriptenv"] != true {
			t.Fatalf("workflow configuration not passed to Go: %s", key)
		}
	}
	foundPrepare, foundSearch, foundChanged := false, false, false
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, child := range v {
				walk(child)
			}
		case map[string]any:
			if v["type"] == "script" {
				args := v["args"].([]any)
				if len(args) != 4 || v["src"] != "./run" {
					t.Fatal("UI must pass query/mode/category as argv")
				}
				if args[0] == "map-ui-search" {
					foundSearch = true
					if v["async"] != true {
						t.Fatal("network request blocks UI")
					}
				}
				if args[0] == "map-ui-prepare" {
					foundPrepare = true
					if v["async"] == true {
						t.Fatal("old request must be invalidated synchronously")
					}
				}
			}
			if v["id"] == "map_query" {
				for _, e := range v["events"].([]any) {
					event := e.(map[string]any)
					if event["triggers"] == "changed" {
						a := event["action"].(map[string]any)
						foundChanged = a["args"].([]any)[0] == "map-ui-prepare"
					}
				}
			}
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(root)
	if !foundPrepare || !foundSearch || !foundChanged {
		t.Fatal("missing prepare/search/invalidation lifecycle")
	}
}

func TestMapUIRunnerFromUnrelatedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "map.alfredui")
	if err := os.Mkdir(ui, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ui, "run"), readFile(t, "../../workflow/map.alfredui/run"), 0755)
	writeFile(t, filepath.Join(root, "run"), []byte("#!/bin/sh\nprintf '%s\\000' \"$@\"\n"), 0755)
	cmd := exec.Command("/bin/bash", filepath.Join(ui, "run"), "map-ui-search", "$100 `literal` 한글", "useIP", "bus")
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "map-ui-search\x00$100 `literal` 한글\x00useIP\x00bus\x00" {
		t.Fatalf("%q: %v", out, err)
	}
}

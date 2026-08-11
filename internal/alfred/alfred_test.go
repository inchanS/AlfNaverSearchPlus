package alfred

import (
	"encoding/json"
	"testing"
)

func TestItemJSON(t *testing.T) {
	fb := New()
	it := fb.Add(ItemOpts{
		Title:        "t",
		Arg:          "a",
		Autocomplete: "ac",
		Copy:         "c",
		LargeType:    "l",
		Icon:         "icon.png",
		Valid:        true,
	})
	it.SetVar("map_type", "place")
	it.SetVar("useIP", false) // must serialize as JSON bool, not "false"

	data, err := json.Marshal(fb)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	item := got["items"].([]any)[0].(map[string]any)

	if item["text"].(map[string]any)["copy"] != "c" {
		t.Error("copytext should map to text.copy")
	}
	if item["icon"].(map[string]any)["path"] != "icon.png" {
		t.Error("icon should map to icon.path")
	}
	vars := item["variables"].(map[string]any)
	if vars["map_type"] != "place" {
		t.Error("map_type variable missing")
	}
	// A bool variable must remain a JSON bool so info.plist conditionals match
	// the previous library's output.
	if v, ok := vars["useIP"].(bool); !ok || v != false {
		t.Errorf("useIP should be JSON bool false, got %#v", vars["useIP"])
	}
}

func TestValidAlwaysEmitted(t *testing.T) {
	fb := New()
	fb.Add(ItemOpts{Title: "x", Valid: false})
	data, _ := json.Marshal(fb)
	// "valid":false must be present (not omitted) so Alfred does not default it.
	if !containsJSONField(data, `"valid":false`) {
		t.Errorf("expected explicit valid:false in %s", data)
	}
}

func containsJSONField(data []byte, field string) bool {
	return len(data) > 0 && json.Valid(data) && contains(string(data), field)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

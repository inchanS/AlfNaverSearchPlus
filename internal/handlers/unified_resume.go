package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Only one search is retained. A row's normal destination identifies the same
// result even if it moves, or was opened with the alternate shopping action.
type lastSearch struct {
	Category string `json:"category"`
	Query    string `json:"query"`
	Selected string `json:"selected,omitempty"`
}

func lastSearchPath() string {
	dir := os.Getenv("alfred_workflow_data")
	if os.Getenv("nn_resume") != "1" || dir == "" {
		return ""
	}
	return filepath.Join(dir, "last-search.json")
}

func (s lastSearch) save() error {
	path := lastSearchPath()
	if path == "" || categoryNames[s.Category] == "" {
		return nil
	}
	if strings.TrimSpace(s.Query) == "" {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".last-search-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(s)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func resumeSearch(initial string) uiJSON {
	path := lastSearchPath()
	if initial != "" || path == "" {
		return uiJSON{}
	}
	data, err := os.ReadFile(path)
	var s lastSearch
	if err != nil || json.Unmarshal(data, &s) != nil || categoryNames[s.Category] == "" || strings.TrimSpace(s.Query) == "" {
		return uiJSON{}
	}
	action := prepareSearch(s.Category, s.Query, "search")
	action["variables"].(uiJSON)["restore_url"] = s.Selected
	ids := action["update"].(uiJSON)["ids"].(uiJSON)
	ids["query"] = uiJSON{"text": "{var:source_query}"}
	return action
}

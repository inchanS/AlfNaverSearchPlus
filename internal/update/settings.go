package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
)

// Store preferences in Alfred's workflow data, which survives cache clearing
// and workflow upgrades. Do not write into the workflow installation itself.
const settingsFile = "update-settings.json"

type updateSettings struct {
	Enabled *bool `json:"automatic_updates,omitempty"`
}

func autoUpdateEnabled() bool {
	dir := os.Getenv("alfred_workflow_data")
	if dir == "" {
		return true
	}
	data, err := os.ReadFile(filepath.Join(dir, settingsFile))
	var settings updateSettings
	if err != nil || json.Unmarshal(data, &settings) != nil || settings.Enabled == nil {
		return true
	}
	return *settings.Enabled
}

func setAutoUpdate(enabled bool) error {
	dir := os.Getenv("alfred_workflow_data")
	if dir == "" {
		return fmt.Errorf("alfred_workflow_data is not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(updateSettings{Enabled: &enabled})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".update-settings-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, settingsFile))
}

// HandleAutoUpdate intercepts the legacy toggle queries before normal search.
// Manual installation remains available regardless of this preference.
func HandleAutoUpdate(query string, fb *alfred.Feedback) bool {
	var enabled bool
	switch query {
	case "workflow:autoupdate":
		enabled = true
	case "workflow:noautoupdate":
		enabled = false
	default:
		return false
	}
	title := "Automatic update checks disabled"
	if enabled {
		title = "Automatic update checks enabled"
	}
	subtitle := "Manual update: n workflow:update"
	if err := setAutoUpdate(enabled); err != nil {
		title = "Could not save automatic update setting"
		subtitle = err.Error()
	}
	fb.Add(alfred.ItemOpts{Title: title, Subtitle: subtitle, Valid: false})
	fb.Send()
	return true
}

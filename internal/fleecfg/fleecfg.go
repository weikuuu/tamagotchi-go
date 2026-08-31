// Package fleecfg persists whether the desktop overlay's "flee from the
// cursor" mode is on. Like overlaycfg, this is read by the overlay process
// and can be written by either the overlay itself (right-click toggle) or
// the main window's settings panel — separate OS processes, synced via
// this file.
package fleecfg

import (
	"os"
	"path/filepath"
	"strings"
)

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "tamagotchi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "flee_mode.txt"), nil
}

// Load returns whether flee mode is enabled. Defaults to false.
func Load() bool {
	p, err := path()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "true"
}

// Save persists whether flee mode is enabled.
func Save(enabled bool) error {
	p, err := path()
	if err != nil {
		return err
	}
	v := "false"
	if enabled {
		v = "true"
	}
	return os.WriteFile(p, []byte(v), 0o644)
}

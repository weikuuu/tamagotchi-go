// Package bubblecfg persists whether the desktop overlay's speech bubble
// is shown at all. Some people find a chatty little sprite charming, others
// just want the pet without the pop-up text — this is the settings-panel
// toggle for that. Read by the overlay process, written by the main
// window's settings panel — separate OS processes, synced via this file.
package bubblecfg

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
	return filepath.Join(dir, "bubble_enabled.txt"), nil
}

// Load returns whether the speech bubble is enabled. Defaults to true.
func Load() bool {
	p, err := path()
	if err != nil {
		return true
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(data)) != "false"
}

// Save persists whether the speech bubble is enabled.
func Save(enabled bool) error {
	p, err := path()
	if err != nil {
		return err
	}
	v := "true"
	if !enabled {
		v = "false"
	}
	return os.WriteFile(p, []byte(v), 0o644)
}

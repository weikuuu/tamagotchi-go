// Package username reads the user's optional display name from a small
// config file, so Elysia can address them by name sometimes. Like
// birthday.txt, there's no in-app UI for it yet — see the README.
package username

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
	return filepath.Join(dir, "username.txt"), nil
}

// Load returns the configured name, or "" if none is set.
func Load() string {
	p, err := path()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Save persists the given name.
func Save(name string) error {
	p, err := path()
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(strings.TrimSpace(name)), 0o644)
}

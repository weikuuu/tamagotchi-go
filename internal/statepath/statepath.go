// Package statepath locates the on-disk file where the elygochi's save
// state lives, shared between the main window and the desktop overlay.
package statepath

import (
	"os"
	"path/filepath"
)

// Default returns the path to the shared save file, creating its parent
// directory if necessary.
func Default() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "elygochi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

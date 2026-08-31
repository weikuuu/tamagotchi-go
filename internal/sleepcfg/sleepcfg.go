// Package sleepcfg tells the desktop overlay when Elysia just went to sleep
// in the main window (after pressing "Спать"), so it can show a matching
// sleepy/yawning sticker instead of whatever her ambient mood art would
// otherwise be. Written by the main window, read by the overlay — separate
// OS processes, synced via this file, same pattern as bubblecfg/fleecfg.
package sleepcfg

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "elygochi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "sleep_until.txt"), nil
}

// Save persists the moment sleepiness should stop being shown.
func Save(until time.Time) error {
	p, err := path()
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(strconv.FormatInt(until.UnixMilli(), 10)), 0o644)
}

// Load returns the persisted "sleepy until" deadline, or the zero Time if
// none is set or the file can't be read.
func Load() time.Time {
	p, err := path()
	if err != nil {
		return time.Time{}
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return time.Time{}
	}
	ms, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

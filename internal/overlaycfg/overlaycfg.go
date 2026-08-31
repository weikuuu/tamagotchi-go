// Package overlaycfg persists the desktop overlay's size, as a scale
// factor applied to its base sprite/bubble/font dimensions. It's read by
// the overlay process and written by the main window's settings panel —
// two separate OS processes, so this goes through disk like the rest of
// the shared config (see internal/statepath).
package overlaycfg

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MinScale and MaxScale bound how small/large the overlay can be made.
const (
	MinScale     = 0.5
	MaxScale     = 2.0
	DefaultScale = 1.0
	Step         = 0.1
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
	return filepath.Join(dir, "overlay_scale.txt"), nil
}

// Load returns the configured scale, or DefaultScale if none is set or the
// file is unreadable/invalid.
func Load() float64 {
	p, err := path()
	if err != nil {
		return DefaultScale
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return DefaultScale
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return DefaultScale
	}
	return Clamp(v)
}

// Save persists the given scale (clamped to [MinScale, MaxScale]).
func Save(scale float64) error {
	p, err := path()
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(strconv.FormatFloat(Clamp(scale), 'f', 2, 64)), 0o644)
}

// Clamp bounds v to [MinScale, MaxScale].
func Clamp(v float64) float64 {
	if v < MinScale {
		return MinScale
	}
	if v > MaxScale {
		return MaxScale
	}
	return v
}

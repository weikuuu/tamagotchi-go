// Package birthday reads and writes Elysia's optional "birthday"
// (day-month) from a small config file, so the app can show birthday
// phrases once a year.
package birthday

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
	return filepath.Join(dir, "birthday.txt"), nil
}

// Load returns the configured birthday as "DD-MM", or "" if none is set.
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

var monthDayPattern = regexp.MustCompile(`^\d{2}-\d{2}$`)

// Save persists the given birthday, which must be "DD-MM".
func Save(dayMonth string) error {
	dayMonth = strings.TrimSpace(dayMonth)
	if !monthDayPattern.MatchString(dayMonth) {
		return fmt.Errorf("birthday: %q is not in DD-MM format", dayMonth)
	}
	p, err := path()
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(dayMonth), 0o644)
}

// IsToday reports whether the configured birthday (if any) matches now.
func IsToday(dayMonth string, now time.Time) bool {
	return dayMonth != "" && dayMonth == now.Local().Format("02-01")
}

// IsNewYear reports whether now is New Year's Day.
func IsNewYear(now time.Time) bool {
	t := now.Local()
	return t.Month() == time.January && t.Day() == 1
}

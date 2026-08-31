// Package birthday reads and writes Elysia's optional "birthday"
// (day-month) from a small config file, so the app can show birthday
// phrases once a year.
package birthday

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

var monthDayPattern = regexp.MustCompile(`^(\d{2})-(\d{2})$`)

// daysInMonth is deliberately permissive about February (allows 29 even
// though not every year is a leap year) since birthdays are stored without
// a year at all.
var daysInMonth = [13]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// Valid reports whether dayMonth is a real calendar date in "DD-MM" form
// (e.g. rejects "67-67").
func Valid(dayMonth string) bool {
	m := monthDayPattern.FindStringSubmatch(strings.TrimSpace(dayMonth))
	if m == nil {
		return false
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	return month >= 1 && month <= 12 && day >= 1 && day <= daysInMonth[month]
}

// Save persists the given birthday, which must be "DD-MM" and a real date.
func Save(dayMonth string) error {
	dayMonth = strings.TrimSpace(dayMonth)
	if !Valid(dayMonth) {
		return fmt.Errorf("birthday: %q is not a valid DD-MM date", dayMonth)
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

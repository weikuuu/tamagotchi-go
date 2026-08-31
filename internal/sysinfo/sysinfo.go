// Package sysinfo reports the name of the frontmost application on macOS,
// used as a lightweight "what's the user up to" activity indicator.
package sysinfo

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

const refreshInterval = 4 * time.Second

// Service periodically polls the frontmost application in the background.
type Service struct {
	mu  sync.RWMutex
	app string
	err error
}

// Start launches a Service and kicks off its first poll immediately.
func Start() *Service {
	s := &Service{}
	go s.loop()
	return s
}

func (s *Service) loop() {
	s.refresh()
	t := time.NewTicker(refreshInterval)
	for range t.C {
		s.refresh()
	}
}

func (s *Service) refresh() {
	app, err := activeApp()
	s.mu.Lock()
	s.app, s.err = app, err
	s.mu.Unlock()
}

// ActiveApp returns the most recently seen frontmost application name, or
// "" if it isn't known yet (or the OS denied Automation permission).
func (s *Service) ActiveApp() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.err != nil {
		return ""
	}
	return s.app
}

const script = `tell application "System Events" to get name of first application process whose frontmost is true`

func activeApp() (string, error) {
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

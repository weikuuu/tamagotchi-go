// Package spotify reports the track currently playing on Spotify, so
// Elysia can react to it. It reads the local desktop app directly
// (spotifylocal) — no login, no developer account, no Client ID. See
// spotifylocal for how each OS does that.
package spotify

import (
	"sync"
	"time"

	"elygochi/internal/spotifylocal"
)

const pollInterval = 2 * time.Second

// Snapshot is the latest known playback state.
type Snapshot struct {
	Track   string
	Artist  string
	Playing bool
	Ready   bool // false until the service has polled at least once successfully
}

// Service polls the local Spotify app in the background. Zero value is
// safe but inert; use Start.
type Service struct {
	mu   sync.RWMutex
	snap Snapshot
}

// Start launches the background poll loop.
func Start() *Service {
	s := &Service{}
	go s.run()
	return s
}

// Snapshot returns the most recently polled playback state.
func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Service) run() {
	poll := func() {
		local := spotifylocal.Get()
		if !local.OK {
			return
		}
		s.mu.Lock()
		s.snap = Snapshot{Track: local.Track, Artist: local.Artist, Playing: local.Playing, Ready: true}
		s.mu.Unlock()
	}
	poll()
	t := time.NewTicker(pollInterval)
	for range t.C {
		poll()
	}
}

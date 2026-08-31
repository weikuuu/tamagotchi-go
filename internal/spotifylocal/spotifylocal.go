// Package spotifylocal reads what's playing straight from the local
// Spotify desktop app instead of the Spotify Web API — no developer app,
// no Client ID, no OAuth, and critically no Premium subscription required
// (the Web API's player-state endpoints are Premium-only; reading the
// desktop client directly isn't).
package spotifylocal

// Snapshot is the current local read. ok is false if Spotify doesn't
// appear to be running or this OS isn't supported, in which case callers
// should fall back to whatever else they have (e.g. the Web API).
type Snapshot struct {
	Track   string
	Artist  string
	Playing bool
	OK      bool
}

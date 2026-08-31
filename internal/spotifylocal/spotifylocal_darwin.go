//go:build darwin

package spotifylocal

import (
	"os/exec"
	"strings"
)

// script asks the local Spotify.app for its player state directly — no
// network, no auth. Returns "||" fields if Spotify isn't running at all;
// osascript's own error in that case is treated the same as "not running"
// rather than surfaced, since that's the overwhelmingly common case (the
// user just doesn't have Spotify open).
const script = `
if application "Spotify" is running then
	tell application "Spotify"
		set trackName to name of current track
		set artistName to artist of current track
		set st to player state as string
		return trackName & "|" & artistName & "|" & st
	end tell
else
	return "not_running"
end if
`

// Get polls the local Spotify.app via AppleScript.
func Get() Snapshot {
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return Snapshot{}
	}
	line := strings.TrimSpace(string(out))
	if line == "" || line == "not_running" {
		return Snapshot{}
	}
	parts := strings.SplitN(line, "|", 3)
	if len(parts) != 3 {
		return Snapshot{}
	}
	return Snapshot{
		Track:   parts[0],
		Artist:  parts[1],
		Playing: parts[2] == "playing",
		OK:      true,
	}
}

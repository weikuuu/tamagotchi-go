//go:build darwin

package spotifylocal

import (
	"os/exec"
	"strings"
)

// script asks the local Spotify.app for its player state directly — no
// network, no auth. Wrapped in "try" because "name of current track" throws
// if Spotify is running but nothing has ever been loaded (fresh launch);
// "player state is playing" is used instead of coercing the state to a
// string, since Spotify's player state is a custom enum and not every
// build accepts that coercion the same way.
const script = `
if application "Spotify" is running then
	tell application "Spotify"
		try
			set trackName to name of current track
			set artistName to artist of current track
			if player state is playing then
				set st to "playing"
			else
				set st to "paused"
			end if
			return trackName & "|" & artistName & "|" & st
		on error
			return "no_track"
		end try
	end tell
else
	return "not_running"
end if
`

// Get polls the local Spotify.app via AppleScript. The first call on a
// fresh install may trigger a one-time macOS "Elygochi wants to control
// Spotify" Automation permission prompt — until that's approved (System
// Settings > Privacy & Security > Automation), osascript fails and this
// just reports OK: false, same as Spotify not running at all.
func Get() Snapshot {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return Snapshot{}
	}
	line := strings.TrimSpace(string(out))
	if line == "" || line == "not_running" || line == "no_track" {
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

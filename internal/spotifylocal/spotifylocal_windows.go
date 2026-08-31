//go:build windows

package spotifylocal

import (
	"strings"
	"syscall"
	"unsafe"
)

// Spotify's Windows desktop client sets its main window's title to
// "Track - Artist" while something is playing, and to a bare "Spotify"
// (or "Spotify Free"/"Spotify Premium") when paused or idle. This has
// been a stable, widely-relied-on behavior for years (it's how most
// Rainmeter/Windows "now playing" widgets read Spotify), and unlike the
// Web API it doesn't care whether the account is Premium or Free.
var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowTextW  = user32.NewProc("GetWindowTextW")
	procGetClassNameW   = user32.NewProc("GetClassNameW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
)

const spotifyWindowClass = "SpotifyMainWindow"

// Get scans top-level windows for Spotify's main window and parses its
// title.
func Get() Snapshot {
	var title string
	found := false

	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1 // continue
		}
		if getClassName(hwnd) != spotifyWindowClass {
			return 1
		}
		title = getWindowText(hwnd)
		found = true
		return 0 // stop enumeration
	})
	procEnumWindows.Call(cb, 0)

	if !found {
		return Snapshot{}
	}
	return parseTitle(title)
}

func getClassName(hwnd uintptr) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

func getWindowText(hwnd uintptr) string {
	buf := make([]uint16, 512)
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

func parseTitle(title string) Snapshot {
	title = strings.TrimSpace(title)
	if title == "" || !strings.Contains(title, " - ") {
		// "Spotify", "Spotify Free", "Spotify Premium", or blank — paused/idle.
		return Snapshot{OK: true}
	}
	parts := strings.SplitN(title, " - ", 2)
	return Snapshot{Track: parts[0], Artist: parts[1], Playing: true, OK: true}
}

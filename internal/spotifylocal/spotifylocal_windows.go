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
// Rainmeter/Windows "now playing" widgets read Spotify).
//
// The window is normally identified by its class name ("SpotifyMainWindow"),
// but that's exactly the kind of internal detail a Spotify update could
// change without notice, so as a second attempt this also matches by the
// owning process's image name (Spotify.exe) if the class-name match finds
// nothing.
var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")

	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
)

const (
	spotifyWindowClass      = "SpotifyMainWindow"
	processQueryLimitedInfo = 0x1000
)

// Get scans top-level windows for Spotify's main window and parses its
// title.
func Get() Snapshot {
	var byClass, byProcess string
	haveClass, haveProcess := false, false

	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1 // continue
		}
		length, _, _ := procGetWindowTextLengthW.Call(hwnd)
		if length == 0 {
			return 1
		}

		if !haveClass && getClassName(hwnd) == spotifyWindowClass {
			byClass = getWindowText(hwnd)
			haveClass = true
			return 0 // stop enumeration — this is the definitive match
		}
		if !haveProcess && strings.EqualFold(processExeName(hwnd), "Spotify.exe") {
			byProcess = getWindowText(hwnd)
			haveProcess = true
			// keep going in case the class-name match still shows up
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)

	if haveClass {
		return parseTitle(byClass)
	}
	if haveProcess {
		return parseTitle(byProcess)
	}
	return Snapshot{}
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

// processExeName returns the base executable filename (e.g. "Spotify.exe")
// of the process that owns hwnd, or "" if it can't be determined.
func processExeName(hwnd uintptr) string {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, _, _ := procOpenProcess.Call(processQueryLimitedInfo, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)

	buf := make([]uint16, 512)
	size := uint32(len(buf))
	ok, _, _ := procQueryFullProcessImageName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return ""
	}
	full := syscall.UTF16ToString(buf[:size])
	if i := strings.LastIndexAny(full, `\/`); i >= 0 {
		return full[i+1:]
	}
	return full
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

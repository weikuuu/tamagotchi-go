// Package autostart toggles launching the app automatically when the user
// logs in — macOS via a LaunchAgent plist, Windows via a Run registry
// value. Linux isn't handled (no autostart toggle shown there); it's a
// desktop-only convenience, not core functionality.
package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const label = "com.tamagotchi.elysia"

// Supported reports whether this OS has an autostart implementation.
func Supported() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}

// Enabled reports whether autostart is currently set up.
func Enabled() bool {
	switch runtime.GOOS {
	case "darwin":
		p, err := agentPath()
		if err != nil {
			return false
		}
		_, err = os.Stat(p)
		return err == nil
	case "windows":
		return windowsEnabled()
	default:
		return false
	}
}

// SetEnabled turns autostart on or off.
func SetEnabled(on bool) error {
	switch runtime.GOOS {
	case "darwin":
		if on {
			return installDarwin()
		}
		return uninstallDarwin()
	case "windows":
		if on {
			return installWindows()
		}
		return uninstallWindows()
	default:
		return fmt.Errorf("autostart: unsupported OS %s", runtime.GOOS)
	}
}

func agentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func installDarwin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Resolve out of the .app bundle's Contents/MacOS to the .app itself
	// isn't necessary — launching the binary directly works fine and is
	// simpler than re-deriving the bundle path.
	p, err := agentPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, label, exe)
	if err := os.WriteFile(p, []byte(plist), 0o644); err != nil {
		return err
	}
	// Best-effort: makes it take effect immediately instead of waiting for
	// the next login. Not fatal if launchctl isn't happy about it.
	_ = exec.Command("launchctl", "load", p).Run()
	return nil
}

func uninstallDarwin() error {
	p, err := agentPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", p).Run()
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

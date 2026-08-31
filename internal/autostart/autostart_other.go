//go:build !windows

package autostart

import "fmt"

func windowsEnabled() bool { return false }

func installWindows() error {
	return fmt.Errorf("autostart: windows-only")
}

func uninstallWindows() error {
	return fmt.Errorf("autostart: windows-only")
}

//go:build windows

package autostart

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const runValueName = "ElysiaTamagotchi"

func windowsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValueName)
	return err == nil
}

func installWindows() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(runValueName, `"`+exe+`"`)
}

func uninstallWindows() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	err = k.DeleteValue(runValueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

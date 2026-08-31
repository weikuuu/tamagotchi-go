// Command elygochi runs the Elysia desktop elygochi: a main window for
// feeding/playing/resting the pet, plus a transparent overlay that lets the
// pet fly around the desktop. The two windows are separate OS processes
// (Ebitengine only supports one window per process) kept in sync through a
// shared save file; running the binary normally starts both.
package main

import (
	"log"
	"os"
	"os/exec"
)

// overlayEnv, when set to "1" in the environment, tells this binary to run
// as the desktop overlay instead of the main window.
const overlayEnv = "TAMAGOTCHI_OVERLAY"

func main() {
	if os.Getenv(overlayEnv) == "1" {
		runOverlay()
		return
	}

	overlay := spawnOverlay()
	runMainWindow()
	if overlay != nil && overlay.Process != nil {
		_ = overlay.Process.Kill()
	}
}

// spawnOverlay launches a second copy of this binary in overlay mode. If it
// fails, the main window still runs on its own.
func spawnOverlay() *exec.Cmd {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("overlay: could not locate executable: %v", err)
		return nil
	}

	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), overlayEnv+"=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("overlay: failed to start: %v", err)
		return nil
	}
	go func() { _ = cmd.Wait() }()
	return cmd
}

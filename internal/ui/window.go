package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenDashboard opens the Crossover dashboard in a dedicated desktop application window
// (using standalone app mode) or falls back to the system's default browser.
func OpenDashboard(port int) {
	url := fmt.Sprintf("http://127.0.0.1:%d/ui/", port)

	switch runtime.GOOS {
	case "linux":
		// Isolate webview session data into user cache
		cacheDir, _ := os.UserCacheDir()
		if cacheDir == "" {
			cacheDir = "/tmp"
		}
		profileDir := filepath.Join(cacheDir, "crossover-ui")
		_ = os.MkdirAll(profileDir, 0755)

		// 1. Try launching in standalone desktop app window (Chrome / Chromium / Edge / Brave)
		candidates := []string{
			"google-chrome",
			"google-chrome-stable",
			"chromium",
			"chromium-browser",
			"brave-browser",
			"microsoft-edge",
		}

		for _, browser := range candidates {
			if path, err := exec.LookPath(browser); err == nil {
				cmd := exec.Command(
					path,
					fmt.Sprintf("--app=%s", url),
					fmt.Sprintf("--user-data-dir=%s", profileDir),
					"--no-first-run",
					"--no-default-browser-check",
					"--class=crossover",
				)
				if err := cmd.Start(); err == nil {
					log.Printf("[ui] Opened dashboard in standalone window via %s", browser)
					return
				}
			}
		}

		// 2. Fallback to system default browser
		cmd := exec.Command("xdg-open", url)
		_ = cmd.Start()

	case "darwin":
		// On macOS, try launching standalone Chrome app window if present
		appCmd := exec.Command("open", "-na", "Google Chrome", "--args", fmt.Sprintf("--app=%s", url))
		if err := appCmd.Run(); err == nil {
			log.Println("[ui] Opened dashboard in standalone window via macOS Chrome")
			return
		}

		// Fallback to default browser (Safari/Chrome)
		cmd := exec.Command("open", url)
		_ = cmd.Start()

	default:
		log.Printf("[ui] Dashboard URL: %s", url)
	}
}

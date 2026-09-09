package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenDashboard opens the OmniDesk dashboard in a dedicated desktop application window
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
		profileDir := filepath.Join(cacheDir, "omnidesk-ui")
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
					"--class=omnidesk",
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
		// Chrome's own single-instance-per-profile forwarding stops a
		// second `--app=` launch from spawning a whole extra OS process,
		// but the forwarded launch still opens an *additional app window*
		// rather than focusing the existing one — so that alone doesn't
		// stop duplicate windows. activateMacDashboardWindow (window_darwin.go)
		// tracks the PID of the window we launched ourselves and, if it's
		// still alive, just brings it forward instead.
		if activateMacDashboardWindow() {
			log.Println("[ui] Focused existing dashboard window")
			return
		}

		// Give the app-mode window its own dedicated Chrome profile, same
		// as the Linux branch above, and launch the binary directly
		// instead of going through `open -na` (which forces Launch
		// Services to spawn a brand new process every time).
		cacheDir, _ := os.UserCacheDir()
		if cacheDir == "" {
			cacheDir = "/tmp"
		}
		profileDir := filepath.Join(cacheDir, "omnidesk-ui")
		_ = os.MkdirAll(profileDir, 0755)

		chromeCandidates := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			filepath.Join(os.Getenv("HOME"), "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
		}
		for _, chromePath := range chromeCandidates {
			if _, err := os.Stat(chromePath); err != nil {
				continue
			}
			cmd := exec.Command(
				chromePath,
				fmt.Sprintf("--app=%s", url),
				fmt.Sprintf("--user-data-dir=%s", profileDir),
				"--no-first-run",
				"--no-default-browser-check",
			)
			if err := cmd.Start(); err == nil {
				recordMacDashboardPID(cmd.Process.Pid)
				log.Println("[ui] Opened dashboard in standalone window via macOS Chrome")
				return
			}
		}

		// Fallback to default browser (Safari/Chrome)
		cmd := exec.Command("open", url)
		_ = cmd.Start()

	case "windows":
		// On Windows, Microsoft Edge is pre-installed on Win 10/11 and supports --app=
		edgePaths := []string{
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			"msedge.exe",
			"msedge",
		}

		for _, p := range edgePaths {
			cmd := exec.Command(p, fmt.Sprintf("--app=%s", url))
			if err := cmd.Start(); err == nil {
				log.Println("[ui] Opened dashboard in standalone window via Microsoft Edge")
				return
			}
		}

		// Fallback to default browser via cmd start
		cmd := exec.Command("cmd", "/c", "start", "", url)
		_ = cmd.Start()

	default:
		log.Printf("[ui] Dashboard URL: %s", url)
	}
}

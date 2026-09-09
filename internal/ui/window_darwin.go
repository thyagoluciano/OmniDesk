//go:build darwin

package ui

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
)

// macDashboardPID tracks the PID of the standalone Chrome window this
// process last launched for the dashboard, so a later OpenDashboard call
// can bring that same window forward instead of opening another one.
var (
	macDashboardMu  sync.Mutex
	macDashboardPID int
)

// recordMacDashboardPID remembers the process we just launched for the
// dashboard window, for activateMacDashboardWindow to find later.
func recordMacDashboardPID(pid int) {
	macDashboardMu.Lock()
	macDashboardPID = pid
	macDashboardMu.Unlock()
}

// activateMacDashboardWindow brings a previously-launched dashboard
// window to the front, if the process recorded for it is still alive,
// instead of letting the caller spawn a new one. Returns false (never
// launched yet, or that window's process has since exited/been closed)
// so the caller falls through to its normal launch path.
func activateMacDashboardWindow() bool {
	macDashboardMu.Lock()
	pid := macDashboardPID
	macDashboardMu.Unlock()

	if pid == 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return false
	}

	script := fmt.Sprintf(`tell application "System Events" to set frontmost of (first process whose unix id is %d) to true`, pid)
	return exec.Command("osascript", "-e", script).Run() == nil
}

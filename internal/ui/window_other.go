//go:build !darwin

package ui

// recordMacDashboardPID and activateMacDashboardWindow are no-ops outside
// macOS — see window_darwin.go for why the darwin branch of OpenDashboard
// needs to track and re-focus the window it launched itself.
func recordMacDashboardPID(pid int) {}

func activateMacDashboardWindow() bool { return false }

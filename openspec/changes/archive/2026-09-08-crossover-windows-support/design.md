## Context

Crossover currently runs as a background service with native tray integration and embedded Webview dashboard on Linux and macOS. See `proposal.md` for motivation to support Windows.

Windows introduces distinct platform requirements:
1. Executable subsystem: Go CLI binaries by default spawn a black console window unless linked with `-H=windowsgui`.
2. Windowing: Microsoft Edge is pre-installed on Windows 10/11 and supports `--app=<url>` standalone mode.
3. Autostart: Managed via the Windows Registry (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`).
4. Installer: Standard Windows executable setup wizard created with Nullsoft Scriptable Install System (NSIS).

## Goals / Non-Goals

**Goals:**
- Compile `crossover.exe` (amd64) configured with `-H=windowsgui` to run silently in the system tray without a console window.
- Implement `internal/installer/windows.go` supporting `crossover install` and `crossover uninstall` via Windows Registry autostart keys.
- Add Windows support to `internal/ui/window.go` launching the dashboard in standalone window mode via `msedge.exe --app=http://127.0.0.1:24850/ui/`.
- Add Windows support to `internal/notify/notify.go` using PowerShell toast notifications.
- Generate high-resolution `assets/crossover.ico` for Windows executables and shortcuts.
- Provide `scripts/crossover.nsi` and `scripts/build-windows.sh` to package `Crossover-Setup-0.1.0-x64.exe` with desktop/start menu shortcuts and uninstaller.

**Non-Goals:**
- Running as a Windows Service (services run in Session 0 and cannot access the user's graphical clipboard or system tray).
- Virtual KVM / mouse redirection (deferred to a future change).

## Decisions

### Decision 1: Binary Subsystem (`-H=windowsgui`)
- **Choice**: Compile Windows release binaries with `-ldflags="-H=windowsgui -s -w"`.
- **Rationale**: Prevents Windows from opening a persistent black prompt box (`cmd.exe`) while Crossover runs in the background.

### Decision 2: Microsoft Edge App-Mode for Dashboard
- **Choice**: Launch `msedge.exe --app=http://127.0.0.1:<port>/ui/` when opening the dashboard on Windows.
- **Rationale**: Edge is guaranteed to be present on Windows 10/11. App-mode displays a clean, dedicated window with Windows native borders and title bar, with zero tabs and zero address bar. Consumes only ~30MB RAM and avoids bundling 150MB of WebView2/Chromium runtimes.

### Decision 3: Per-User Installation via NSIS
- **Choice**: Target `%LOCALAPPDATA%\Programs\Crossover\` in the NSIS installer.
- **Rationale**: Does not require Administrator elevation (UAC prompt), installs in 2 seconds, and works for any user on standard Windows installations.

### Decision 4: Registry Autostart (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`)
- **Choice**: Register `crossover.exe daemon` under current user registry.
- **Rationale**: Standard Windows mechanism for tray applications (like Discord, Slack, Spotify). Easily inspected and toggled in Windows Task Manager > Startup Apps.

## Risks / Trade-offs

- **[Risk]** Windows Defender Firewall blocking port 24850 on private networks.
  - **Mitigation**: NSIS setup and CLI documentation advise users to click "Allow access" (Permitir) on Private Networks when prompted on first run.
- **[Risk]** NSIS compiler (`makensis`) availability on Linux build host.
  - **Mitigation**: `scripts/build-windows.sh` always produces the standalone `crossover.exe` and a portable `.zip`. If `makensis` is present, it additionally compiles `Crossover-Setup-0.1.0-x64.exe`.

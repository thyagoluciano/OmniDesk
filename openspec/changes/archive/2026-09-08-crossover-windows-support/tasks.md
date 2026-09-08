## 1. Windows Integration & UI Launcher

- [x] 1.1 Update `internal/ui/window.go` to support Windows Edge app mode (`msedge --app=...`) with fallback to default browser, verified by inspecting implementation.
- [x] 1.2 Update `internal/notify/notify.go` to dispatch Windows notifications via PowerShell toast command, verified by inspecting notify implementation.
- [x] 1.3 Generate Windows icon file `assets/crossover.ico` from PNG assets, verified by verifying file creation and size.

## 2. Windows Installer CLI & Autostart Management

- [x] 2.1 Implement `internal/installer/windows.go` to manage `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` registry autostart for `crossover install` and `crossover uninstall`, verified by code verification.
- [x] 2.2 Wire Windows installer in `cmd/crossover/main.go` for `runtime.GOOS == "windows"`, verified by dry-run CLI test.

## 3. NSIS Setup Wizard & Build Automation

- [x] 3.1 Create NSIS installer script `scripts/crossover.nsi` configured with Modern UI 2, shortcuts, and uninstaller, verified by inspecting script syntax.
- [x] 3.2 Create build automation script `scripts/build-windows.sh` to compile `crossover.exe` (`-H=windowsgui`) and generate the NSIS setup wizard `Crossover-Setup-0.1.0-x64.exe` (and portable `.zip`), verified by running the script.

## 4. End-to-End Verification

- [x] 4.1 Cross-compile Windows binary `crossover.exe` and test binary format with `file crossover.exe`.
- [x] 4.2 Execute `scripts/build-windows.sh` and verify generated Windows artifacts in `dist/`.

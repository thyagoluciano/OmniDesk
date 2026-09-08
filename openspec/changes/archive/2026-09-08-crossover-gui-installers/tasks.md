## 1. Desktop Webview Frontend & Embedded Serving

- [x] 1.1 Create modern HTML5/CSS/JS frontend in `web/` with responsive peer cards, drag-and-drop dropzones, real-time status polling, and PIN pairing modal, verified by inspecting asset files.
- [x] 1.2 Wire `embed.FS` and register the `/ui/` static file server route in `internal/core/server.go`, verified by curling `http://127.0.0.1:24850/ui/` and receiving the HTML payload.
- [x] 1.3 Implement desktop window launcher in `internal/ui/window.go` (Webview with automatic browser fallback) and connect it to the system tray "Open Dashboard" action, verified by running `crossover gui` or triggering tray click.

## 2. Linux Installer & Autostart Integration

- [x] 2.1 Create Freedesktop `.desktop` template and high-resolution application icon under `assets/`, verified by checking file existence and format.
- [x] 2.2 Implement `crossover install` and `crossover uninstall` CLI commands in `internal/installer/linux.go` and `cmd/crossover/main.go` for binary, desktop launcher, and icon placement, verified by executing `crossover install --help`.
- [x] 2.3 Implement systemd user service generation (`crossover.service`) and automatic enablement via `systemctl --user`, verified by unit file syntax check and systemd service status.
- [x] 2.4 Create Debian packaging script `scripts/build-deb.sh`, verified by running the script and testing package inspection with `dpkg-deb -c`.

## 3. macOS Installer & Autostart Integration

- [x] 3.1 Create macOS application bundle script `scripts/build-macos-app.sh` with `Contents/Info.plist` and resource structuring, verified by checking the generated `Crossover.app` directory structure.
- [x] 3.2 Implement macOS `LaunchAgent` plist template and management in `internal/installer/darwin.go`, verified by inspecting the generated plist XML structure.
- [x] 3.3 Create Apple Disk Image packager `scripts/build-dmg.sh` with drag-to-Applications symlink, verified by running the script.

## 4. End-to-End Build & Verification

- [x] 4.1 Build and test Linux executable and installer commands on the local Ubuntu desktop, verifying the GUI dashboard and tray integration.
- [x] 4.2 Cross-compile macOS Darwin executable and verify bundle creation script execution.

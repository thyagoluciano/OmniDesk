## Context

Crossover currently operates as a background daemon with CLI commands and a system tray icon (`internal/ui/systray.go`). It exposes an internal HTTP REST API on port 24850 for peer status, clipboard synchronization, and file streaming. See `proposal.md` for motivation.

To provide a visual experience for non-technical users and quick drag-and-drop file transfers, we introduce a native desktop GUI and native system installers for both Ubuntu Linux and macOS.

## Goals / Non-Goals

**Goals:**
- Provide a responsive, modern desktop GUI running inside a native Webview window (WebKitGTK on Linux, WKWebView on macOS) with zero Electron bloat.
- Embed frontend static assets (HTML5/CSS/JS) directly into the Go binary using `embed.FS`.
- Enable drag-and-drop file transfers directly onto visual peer cards with real-time upload progress indicators.
- Provide real-time peer discovery updates and interactive PIN pairing dialogues within the GUI.
- Provide `crossover install` and `crossover uninstall` CLI commands on Linux to manage `~/.local/bin`, `.desktop` entry, application icons, and `systemd --user` autostart.
- Provide macOS bundle packaging (`Crossover.app`), `LaunchAgent` plist autostart configuration, and a `.dmg` creation script.
- Ensure graceful fallback: if native Webview libraries are absent or fail to initialize, automatically open the embedded dashboard in the default browser.

**Non-Goals:**
- Using Electron or Chromium Embedded Framework (CEF), which would bloat the binary by >150MB.
- Remote/cloud relay servers: Crossover remains strictly peer-to-peer over local network / Wi-Fi.
- Re-architecting the core network protocols or clipboard sync engines.

## Decisions

### Decision 1: Embedded Frontend with Native Webview & Browser Fallback
- **Choice**: Serve dashboard HTML/CSS/JS from Go's embedded filesystem (`embed.FS`) on the local daemon port (e.g. `http://127.0.0.1:24850/ui/`). Render it in a native Webview window (`github.com/webview/webview_go` or lightweight system window). If native WebKit bindings are not available, fallback to opening the URL with the OS default browser (`xdg-open` on Linux, `open` on macOS).
- **Rationale**: Webview keeps binary size under 30MB, RAM usage under 35MB, and requires zero Node.js/npm dependencies during build. Browser fallback ensures Crossover still works even on minimal headless/server distributions.
- **Alternatives considered**:
  - *Electron*: Bloated (150MB+ bundle, high RAM usage).
  - *Fyne/Gio (Pure Go UI)*: Complex custom rendering, inconsistent drag-and-drop file handling across diverse Linux window managers, and rigid styling compared to modern CSS.

### Decision 2: Frontend Tech Stack
- **Choice**: Modern Vanilla HTML5, Tailwind CSS (bundled standalone), and standard ES6 JavaScript.
- **Rationale**: No external build pipeline (no Vite/Webpack/npm required to build Go binaries). Can be compiled directly with `go build`.
- **Drag-and-Drop**: Leverages standard HTML5 Drag and Drop events (`dragover`, `dragleave`, `drop`) on peer cards, sending files via standard `multipart/form-data` POST to `/api/v1/transfer`.

### Decision 3: Linux Installation Architecture
- **Choice**: User-level installation into standard XDG paths:
  - Binary: `~/.local/bin/crossover`
  - Desktop launcher: `~/.local/share/applications/crossover.desktop`
  - App Icon: `~/.local/share/icons/hicolor/256x256/apps/crossover.png`
  - Autostart: `~/.config/systemd/user/crossover.service` enabled via `systemctl --user enable --now crossover.service`
  - Distribution: Helper script `scripts/build-deb.sh` to package these into a standard `.deb` using `dpkg-deb`.
- **Rationale**: No root (`sudo`) required for standard user installation. Complies with Freedesktop and modern systemd user session management.

### Decision 4: macOS Installation Architecture
- **Choice**: Standard `Crossover.app` bundle:
  - `Crossover.app/Contents/MacOS/crossover`
  - `Crossover.app/Contents/Info.plist` (with `LSUIElement` / background capability flags)
  - `Crossover.app/Contents/Resources/AppIcon.icns`
  - Autostart: `~/Library/LaunchAgents/com.crossover.app.plist` loaded via `launchctl`.
  - Distribution: Helper script `scripts/build-dmg.sh` utilizing Apple `hdiutil` to generate a drag-to-Applications `.dmg`.
- **Rationale**: Follows Apple Human Interface Guidelines and standard macOS software distribution conventions.

## Risks / Trade-offs

- **[Risk]** Systemd user service losing access to X11/Wayland clipboard display.
  - **Mitigation**: Specify `Environment="DISPLAY=:0"` and `PartOf=graphical-session.target` in `crossover.service`, ensuring the service runs within the active graphical user session.
- **[Risk]** Cross-compiling Webview CGO from Linux for macOS Darwin.
  - **Mitigation**: Use Go build tags to isolate Webview invocation. macOS binaries compiled without CGO fall back seamlessly to opening the embedded dashboard in Safari/default browser upon tray click, while native macOS compilation links WKWebView.

# OmniDesk Product Roadmap

Welcome to the **OmniDesk Product Roadmap**! This document outlines our product vision, milestones, upcoming features, and technical goals.

Our mission is to build the fastest, most reliable, and completely private cross-platform tool for local-network productivity — unifying clipboard, files, and peripherals across all your computers without relying on cloud servers.

---

## Status Legend

| Symbol | Status | Description |
| :---: | :--- | :--- |
| :white_check_mark: | **Completed** | Feature is implemented, tested, and released. |
| :construction: | **In Progress** | Actively under development in the current cycle. |
| :calendar: | **Planned** | Scoped and queued for upcoming milestones. |
| :bulb: | **Under Consideration** | Under evaluation; feedback and RFCs welcome from the community. |

---

## Roadmap Milestones

```
  v0.1.x (Foundation)   ──►   v0.2.x (KVM & Input)   ──►   v0.3.x (Security & Network)
  [Clipboard & Files]         [Mouse & Keyboard]           [mTLS & Smart Filters]
                                                                     │
                                                                     ▼
  v1.0.0 (General Availability) ◄─── v0.4.x (Audio & Headless) ◄─────┘
  [Mobile & Auto-Updates]            [Media & HomeLab]
```

---

### Phase 1: Core Foundation & Local Sync (`v0.1.x` - Current)
*Goal: Provide a rock-solid, zero-cloud foundation for local clipboard synchronization and high-speed file transfer.*

- [x] :white_check_mark: **100% Local P2P Architecture**: Pure local socket communication with zero cloud dependencies.
- [x] :white_check_mark: **Real-Time Clipboard Sync**: Instant bidirectional text and URL synchronization across Linux, macOS, and Windows.
- [x] :white_check_mark: **Infinite Loop Prevention**: Smart hash comparison and deduplication to prevent clipboard echo loops.
- [x] :white_check_mark: **Direct File Transfer Engine**: Chunked P2P file streaming with drag-and-drop web interface and CLI tool (`omnidesk send`).
- [x] :white_check_mark: **Automated Peer Discovery**: Zero-configuration discovery using mDNS (ZeroConf) and local subnet scanning.
- [x] :white_check_mark: **Secure Mutual Pairing**: Temporary 6-digit PIN verification and cryptographically signed pairing tokens.
- [x] :white_check_mark: **Modern Web Dashboard**: Responsive dashboard at `http://127.0.0.1:24850/ui/` with real-time peer status and transfer monitoring.
- [x] :white_check_mark: **System Tray & Native Services**: Native system tray integration and automated service installers (Linux `systemd`, macOS `launchd`, Windows Service).
- [x] :white_check_mark: **Localhost Management Security**: Restrict administrative endpoints exclusively to the local loopback interface.

---

### Phase 2: Input Sharing & KVM Evolution (`v0.2.x` - In Progress)
*Goal: Transform OmniDesk into a software KVM, allowing seamless mouse and keyboard sharing across multiple computers on your desk.*

- [x] :white_check_mark: **Input Sharing Core Engine**: Low-latency binary input event protocol (mouse move, button click, scroll, key down/up).
- [x] :white_check_mark: **Linux X11 Input Capture & Emulation**: X11 RECORD extension capture and XTEST event synthesis.
- [ ] :construction: **Linux Wayland Support**: Integration with Wayland Input Capture via `libei` and Desktop Portals.
- [ ] :construction: **macOS Input Capture & Synthesis**: Support for CoreGraphics event taps and synthetic event injection.
- [ ] :calendar: **Windows Input Sharing**: Low-level Windows hooks (`WH_MOUSE_LL`, `WH_KEYBOARD_LL`) and `SendInput` simulation.
- [ ] :calendar: **Visual Multi-Monitor Screen Arrangement**: Interactive drag-and-drop display layout editor in the Web UI to configure physical monitor positioning (Left, Right, Above, Below).
- [ ] :calendar: **Seamless Cursor Edge Transition**: Fluid cursor handoff between adjacent physical screens with configurable edge friction.
- [ ] :calendar: **Toggle Hotkeys**: Configurable global hotkey (e.g., `ScrollLock` or `Ctrl+Alt+O`) to switch input control manually.
- [ ] :calendar: **Clipboard History & Search**: Local SQLite/file-backed clipboard history buffer with quick search and favorite snippets.

---

### Phase 3: Security Hardening & Advanced Networking (`v0.3.x`)
*Goal: Deliver enterprise-grade encryption and granular networking controls for power users and sensitive environments.*

- [ ] :calendar: **End-to-End Encryption (mTLS / Noise Protocol)**: Upgrade peer-to-peer transport to mutual TLS or Noise IK handshake for forward secrecy across all data streams.
- [ ] :calendar: **Network Interface Binding**: Allow users to bind OmniDesk to specific network adapters (e.g., dedicated Ethernet, Wi-Fi, or WireGuard / Tailscale VPN interfaces).
- [ ] :calendar: **Smart Clipboard Privacy Filters**:
  - Automatically ignore passwords copied from password managers (Bitwarden, 1Password, KeePass).
  - Regex rules to redact or skip credit card numbers, API keys, and sensitive tokens.
- [ ] :calendar: **Resumable Transfers & Checksum Verification**: Checkpoint-based resumption of interrupted transfers with SHA-256 integrity verification.
- [ ] :calendar: **Folder Transfer & Compressed Streaming**: Send entire directories with on-the-fly streaming tar/zip compression without creating temporary archive files on disk.

---

### Phase 4: Audio & Extended Ecosystem (`v0.4.x`)
*Goal: Broaden OmniDesk capabilities to media streaming and headless server environments.*

- [ ] :bulb: **Low-Latency Audio Sharing**: Stream system audio or microphone input between paired computers with minimal latency (Opus codec).
- [ ] :calendar: **Headless Daemon Mode**: Standalone, lightweight binary for headless Linux boxes, home labs, and Raspberry Pi with CLI-first controls.
- [ ] :calendar: **UNIX Pipe CLI Integration**: Stream files directly via terminal pipes:
  ```bash
  cat report.pdf | omnidesk send MacBook-Pro
  ```
- [ ] :bulb: **Customizable Notification Rules**: Granular notification settings per device and event type.

---

### Phase 5: General Availability & Mobile Apps (`v1.0.0+`)
*Goal: Achieve full multi-device ubiquity across desktop and mobile operating systems.*

- [ ] :bulb: **Mobile Companion Apps (Android & iOS)**:
  - Share clipboard text and links between phones, tablets, and desktops.
  - Quick camera roll photo and file sharing to desktop.
- [ ] :calendar: **Automated Background Updates**: Self-updating binary with release channel support (Stable, Beta, Nightly).
- [ ] :bulb: **Community Plugin System**: Webhook and gRPC/JSON-RPC API for community extensions and home automation integrations.

---

## Proposing Features & Participating in the Roadmap

We welcome input from our community! Here is how you can help shape the future of OmniDesk:

1. **Discuss Ideas**: Open a thread in [GitHub Discussions](https://github.com/thyagoluciano/OmniDesk/discussions) under the **Ideas** category to get early feedback.
2. **Submit a Request**: Use the [Feature Request Form](https://github.com/thyagoluciano/OmniDesk/issues/new?template=feature_request.yml).
3. **Submit an RFC**: For major architecture changes or protocol revisions, submit an RFC document via a Pull Request.
4. **Vote**: React with :thumbsup: on existing issues and discussions to help maintainers prioritize work.

*Note: This roadmap is a living document and subject to change based on community feedback, technical feasibility, and security considerations.*

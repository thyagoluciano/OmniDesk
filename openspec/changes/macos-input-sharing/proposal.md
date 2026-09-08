## Why

The KVM-style input-sharing feature (`kvm-input-sharing`) shipped its protocol, transport, permission model, screen-layout graph, and dashboard fully OS-agnostic, but the only platform backend implemented so far is Linux/X11 (`internal/inputshare/capture_linux.go`). `design.md`'s own roadmap deferred macOS (and Windows, and Wayland) to later phases, each meant to be proposed as its own OpenSpec change when started. This is that change for macOS: without it, a Mac can pair, sync clipboard, and transfer files with a paired node, but hits `ErrUnsupportedPlatform` the moment input-sharing tries to start — no capture, no injection, no edge-crossing. The user's actual driver: testing the real two-machine KVM flow between a MacBook and a Linux machine, which today's Linux-only backend cannot do at all.

## What Changes

- Add a macOS platform backend implementing the existing `inputshare.Backend` interface (`internal/inputshare/capture.go`) behind a `darwin` build tag, mirroring `capture_linux.go`'s shape: global capture via `CGEventTap`, injection via `CGEventPost`, `ScreenRect` from the main display, and `Suppress`/`Release` (added to the interface this session for X11's `XGrabPointer`/`XGrabKeyboard`) implemented via the event tap's own consume-vs-pass-through return value — the platform-native equivalent of a grab, not a new mechanism.
- Add a macOS keycode ⇄ HID Usage Code translation table (`keymap_darwin.go`), mirroring `keymap_linux.go`'s pattern: native macOS virtual keycode → canonical HID Usage Code on capture, HID Usage Code → macOS virtual keycode on injection. Never transmits an interpreted character, matching the existing cross-platform rule in `kvm-input-sharing`'s `design.md`.
- Formally resolve the cgo-vs-`purego` decision `kvm-input-sharing/design.md` deferred to this phase (Open Question, section "cgo: decisão adiada para a fase específica de macOS"). `internal/ui/systray_darwin_nocgo.go` is the project's existing precedent for avoiding cgo on darwin; this decision is written up in `design.md` with its actual tradeoffs for `CGEventTap`/`CGEventPost` specifically (a C callback is required for the tap, which constrains how far a no-cgo approach can go — the decision must confront this directly, not just default to the existing precedent).
- Add first-run onboarding for the two macOS permissions input-sharing requires (Accessibility, for `CGEventPost`; Input Monitoring, for `CGEventTap`): detect the missing grant, explain what's needed and why, and guide the user to the correct System Settings pane — there is no existing runtime-permission-check precedent in `internal/installer/darwin.go` (it only handles LaunchAgent install and Gatekeeper/codesign) to reuse, so this is new.
- **New capability** `input-capture-inject-macos`, sibling to `kvm-input-sharing`'s `input-capture-inject-x11` — same shape of requirements (global capture regardless of focused app, indistinguishable injection, keycode↔HID translation both ways, local-suppression while sending), one macOS-specific requirement added for the permission-onboarding gap above.

**Out of scope for this change:** Windows and Linux/Wayland backends (separate future phases per the existing roadmap); any change to the already-shipped protocol, transport, permission model, screen-layout graph, or dashboard — this change is purely "give macOS the same platform backend Linux already has," reusing everything else as-is.

## Capabilities

### New Capabilities
- `input-capture-inject-macos`: global mouse/keyboard capture and injection on macOS via CGEventTap/CGEventPost, HID Usage Code translation, local-input suppression while sending, and first-run Accessibility/Input Monitoring permission onboarding.

### Modified Capabilities
(none — this change only adds a new platform backend; it does not change any existing capability's requirements)

## Impact

- **New files**: `internal/inputshare/capture_darwin.go` (backend, `darwin` build tag), `internal/inputshare/keymap_darwin.go` (keycode↔HID table).
- **Permission onboarding**: new code path, likely `internal/installer/darwin.go` or a new `internal/inputshare/permission_darwin.go`, invoked from the dashboard/tray when input-sharing fails to start due to a missing grant (design.md's own risk register already names this: "[Risco] macOS exige permissão de Acessibilidade + Monitoramento de Entrada... Mitigação (fase 4): fluxo de onboarding claro...").
- **Build/dependency impact**: depends entirely on the cgo-vs-purego decision this change's `design.md` makes — cgo changes the build (CGO_ENABLED=1, a C compiler required for macOS cross-builds/CI), purego changes which macOS system libraries get resolved via `dlopen`/`dlsym` at runtime instead of link time. Either way, no change to any other platform's build.
- **No transport/protocol/wire-format changes** — the macOS backend speaks the exact same `inputshare.Event` wire protocol already in production for X11.

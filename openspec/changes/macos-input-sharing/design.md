## Context

`kvm-input-sharing`'s `Backend` interface (`internal/inputshare/capture.go`) is already fully OS-agnostic:

```go
type Backend interface {
	Name() string
	Start(ctx context.Context, cb Callbacks) error
	Stop()
	Inject(ev Event) error
	ScreenRect() (ScreenRect, error)
	Suppress() error
	Release() error
}
```

`capture_linux.go` implements it for X11 (RECORD for capture, XTest for injection, `XGrabPointer`/`XGrabKeyboard` for `Suppress`/`Release`). `newPlatformBackend` (in `capture.go`) is the single build-tag-selected hook every platform registers into — this change adds `capture_darwin.go`, registering the same way, so `Manager` and everything above it (protocol, transport, permission, layout, dashboard) need zero changes.

The macOS-native surface for this is CoreGraphics's Quartz Event Services: `CGEventTapCreate` for global capture, `CGEventPost` for injection. Both live in `ApplicationServices.framework` (specifically its `HIServices` sub-framework); the run loop machinery they depend on (`CFMachPortCreateRunLoopSource`, `CFRunLoopAddSource`, `CFRunLoopRun`) is in `CoreFoundation.framework`. Two TCC-gated permissions guard this API: **Accessibility** (required for `CGEventPost` to actually affect other apps) and **Input Monitoring** (required for `CGEventTapCreate` to see events at all, introduced macOS 10.15+). `internal/installer/darwin.go` today only handles LaunchAgent install and Gatekeeper/codesign — there's no existing runtime permission-check code to build on.

## Goals / Non-Goals

**Goals:**
- Implement `Backend` for macOS with the exact same observable behavior X11 already has: global capture regardless of focused app, injection indistinguishable from physical input, HID Usage Code translation both directions, and `Suppress`/`Release` around sending.
- Resolve the cgo-vs-purego question this phase inherited as an open question from `kvm-input-sharing/design.md`, with a real decision and rationale — not another deferral.
- First-run onboarding when Accessibility/Input Monitoring aren't granted yet, clear enough that a non-technical user can complete it without external docs.

**Non-Goals:**
- Windows or Linux/Wayland backends (separate future phases).
- Any change to the wire protocol, transport, permission model, or screen-layout graph — the macOS backend is a drop-in `Backend` implementation, nothing upstream of it changes.
- Multi-monitor-aware layout on macOS specifically — `ScreenRect` reports the same "one rectangle for the whole desktop" model every other backend uses (`kvm-input-sharing/design.md` Decision 5), including when multiple displays are arranged in macOS's own Display settings.

## Decisions

### 1. purego, not cgo — resolving the deferred question

**Decision: purego.** `internal/ui/systray_darwin_nocgo.go` already establishes this project's preference for avoiding cgo on darwin, and unlike that file's tray-icon case (where the no-cgo fallback just does *less* — no tray, daemon-only), purego is not a degraded fallback here: it can do the *entire* job.

The one part of this API that looks cgo-shaped is `CGEventTapCreate`'s callback: it's declared as a plain C function pointer (`CGEventTapCallBack`), and the OS calls it synchronously from inside a running `CFRunLoop`. That's exactly the case `purego.NewCallback` exists for — wrapping a Go function as a C-callable function pointer — so the callback isn't actually a blocker. What purego *does* require here that cgo wouldn't: the capture goroutine must call `runtime.LockOSThread()` and physically drive `CFRunLoopRun()` on that thread itself (via `purego.RegisterLibFunc` against `CoreFoundation.framework`, `dlopen`'d) instead of getting a C runtime that does it implicitly — bookkeeping, not a capability gap.

What actually tips this: **cross-compilation.** A cgo darwin build needs a real macOS SDK and a C compiler targeting darwin at build time — from a Linux CI worker or a Linux dev machine (this project's actual daily build environment, per this session), that means a full macOS cross-toolchain (osxcross or equivalent) or an actual Mac in the build pipeline. A purego darwin build is `GOOS=darwin GOARCH=arm64 go build`, no C toolchain, no SDK, from anywhere — everything Apple-side is resolved by `dlopen`/`dlsym` **at runtime on the Mac that actually runs the binary**, not at build time. Given today's Linux+Mac cross-machine testing already showed how much friction cross-building/deploying introduces, this is a real, not theoretical, win.

**Alternative considered — cgo:** simpler code at the call site (write the callback as a real C function, no `NewCallback` indirection; CFRunLoop integration is what `import "C"` + a tiny embedded C shim would give you for free). Rejected: it inherits the standard Go cgo costs (slower builds, the cross-compilation problem above, CGO_ENABLED=1 as a hard requirement) for a benefit — slightly less indirect callback plumbing — that doesn't offset them, and it breaks with this project's own established darwin precedent for no clear compensating reason.

### 2. Frameworks and symbols, dlopen'd via purego

Two frameworks, loaded once at backend `Start`:
- `/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices`: `CGEventTapCreate`, `CGEventTapEnable`, `CGEventPost`, `CGEventCreateKeyboardEvent`, `CGEventCreateMouseEvent`, `CGEventCreateScrollWheelEvent`, `CGEventGetIntegerValueField`, `CGEventSetIntegerValueField`, `CGMainDisplayID`, `CGDisplayPixelsWide`/`CGDisplayPixelsHigh` (for `ScreenRect`), `AXIsProcessTrustedWithOptions` (Accessibility permission check/prompt).
- `/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation`: `CFMachPortCreateRunLoopSource`, `CFRunLoopGetCurrent`, `CFRunLoopAddSource`, `CFRunLoopRun`, `CFRunLoopStop`, `CFRelease`, plus the handful of `CFString`/`CFDictionary` helpers `AXIsProcessTrustedWithOptions`'s options dictionary needs.

These are system frameworks present on every macOS install at fixed paths back to 10.x — no bundled dependency, no version pinning concern.

### 3. Suppress/Release: a callback flag, not a separate grab call

X11's `Suppress` issues a distinct `XGrabPointer`/`XGrabKeyboard` request; `Release` un-grabs. On macOS there is no separate grab primitive to call — `CGEventTapCreate` is created once, at backend `Start`, with `kCGEventTapOptionDefault` (the mode that allows the callback to consume events, vs. `kCGEventTapOptionListenOnly`). The callback itself decides per-event whether to pass the event through (return it unchanged) or swallow it (return `NULL`, which is exactly what `XGrabPointer`'s empty event mask achieves for X11 — the event reaches no other app). `Suppress`/`Release` become nothing more than flipping an `atomic.Bool` the callback reads on every invocation; no CGEventTap re-creation, no extra round-trip. This is simpler than the X11 path, not just a port of it.

### 4. Keymap: literal table, not a formula

`keymap_linux.go` derives X11 keycodes from evdev codes with a fixed `+8` offset — X11's keycode space is evdev's, shifted by a constant. macOS virtual keycodes (`kVK_ANSI_A = 0x00`, etc., historically from `Carbon/HIToolbox`, now just documented constants with no supported public header to import without Carbon) have no such formula relative to HID Usage Codes; `keymap_darwin.go` is a literal `[]struct{ VK byte; HID HIDUsage }` table, hand-built the same way `evdevToHID` was, covering the same key set `keymap_linux.go` covers today (letters, digits, standard punctuation, modifiers, function keys, arrows, common editing/navigation keys). Keys with no mapping are dropped with a logged warning at capture time, per the existing cross-platform rule (`kvm-input-sharing/design.md`: never transmit an interpreted character, and a backend must skip an untranslatable key rather than send a zero value).

### 5. Permission onboarding: detect via the real API call, not a heuristic

`AXIsProcessTrustedWithOptions` (with the "prompt" option set) is the one call that both checks Accessibility *and* can trigger the system's own "OmniDesk wants to control this computer" prompt — used at backend `Start`. Input Monitoring has no equivalent public check API; the practical signal is `CGEventTapCreate` itself returning `NULL`. So the flow is: check Accessibility via `AXIsProcessTrustedWithOptions` first (prompting if not yet granted); if `CGEventTapCreate` still fails afterward, treat that as "Input Monitoring also needed" and surface both System Settings panes (Privacy & Security → Accessibility, and → Input Monitoring) in one onboarding message, since the OS gives no clean way to tell which of the two specifically blocked it. `Backend.Start` returns a descriptive error in this case (matching `capture_linux.go`'s existing pattern of degrading to "no backend" with a logged reason rather than crashing); the dashboard/tray surfaces it as an actionable message instead of a silent no-op.

## Risks / Trade-offs

- **[Risk] `purego.NewCallback`'s callback runs on an arbitrary OS thread the run loop was started on, and must never block** → Mitigation: the tap callback only translates the event and hands it to the existing `Callbacks` struct (already designed to be cheap — X11's `dispatchMotion`/`dispatchKey` do the same); no network I/O or lock contention inside the callback itself.
- **[Risk] `CGEventTapCreate` can silently stop delivering events if the callback is too slow (the OS disables a tap that misses its budget) — a known macOS behavior, not specific to this design** → Mitigation: `CGEventTapEnable` re-arms on a `kCGEventTapDisabledByTimeout` notification (a case `CGEventTapCreate`'s callback signature already reports); handle it explicitly rather than leaving capture silently dead — this is the macOS analog of the X11 stream-desync risk `kvm-input-sharing` hit and fixed for real in `record_conn_linux.go`, worth taking seriously from the start here instead of discovering it the same way.
- **[Risk] Users on macOS 10.14 or older lack the Input Monitoring permission entirely (it didn't exist before 10.15)** → Mitigation: not handled specially — `CGEventTapCreate` behavior on those older versions only needs Accessibility, so the same code path works, it just never hits the "also need Input Monitoring" branch. No version-gating needed.
- **[Trade-off] purego's dlopen/dlsym happens at runtime, so a typo'd symbol name fails at `Start()` time, not at build time (cgo would catch it at compile time via header declarations)** → Accepted: covered by exercising `Start` (and its failure path) directly rather than relying on the build to catch it — the same category of runtime-vs-compile-time tradeoff every purego consumer accepts, and consistent with what `systray_darwin_nocgo.go` already accepted for its own purego-free (there, cgo-free-and-API-free) approach.

## Migration Plan

Additive only — a new build-tag-gated file, no changes to any shared code path. Ships when `capture_darwin.go` lands and `newPlatformBackend` picks it up on `darwin`; a Mac that previously hit `ErrUnsupportedPlatform` starts working, nothing else changes behavior. No flag, no rollback beyond reverting the new files if something's wrong with them.

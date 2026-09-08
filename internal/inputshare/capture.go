package inputshare

import (
	"context"
	"fmt"
)

// Callbacks are invoked by a platform Backend whenever it captures a local
// input event. They fire regardless of whether this node currently owns
// input locally or is actively sending to a peer — the Manager decides what
// to do with each event (forward it, use it for border-crossing detection,
// or ignore it while receiving from a peer).
type Callbacks struct {
	// OnMotion fires on pointer movement. absX/absY are the cursor's
	// absolute position in the local screen's pixel space (used for
	// border-crossing detection); dx/dy are the relative delta since the
	// previous motion event (used for the wire MouseMoveEvent once a
	// session is active).
	OnMotion func(absX, absY int, dx, dy int16)
	OnButton func(btn MouseButton, pressed bool)
	OnScroll func(dx, dy int16)
	// OnKey fires with the key already translated to its canonical HID
	// usage code. A backend that cannot translate a captured key code MUST
	// skip it (log a warning) rather than call OnKey with a zero value.
	OnKey func(hid HIDUsage, pressed bool)
}

// Backend is the platform-specific half of input sharing: global capture of
// local mouse/keyboard events and injection of remote ones. Each OS ships
// its own implementation behind a build tag (see capture_linux.go); this
// change only implements Linux/X11 (design.md roadmap, phase 1).
type Backend interface {
	// Name identifies the backend for logs, e.g. "x11".
	Name() string
	// Start begins global capture, delivering events to cb until ctx is
	// canceled or Stop is called. Must return once the capture loop has
	// actually started (or failed).
	Start(ctx context.Context, cb Callbacks) error
	// Stop ends capture and releases any OS resources.
	Stop()
	// Inject synthesizes a local input event equivalent to the one
	// described by ev (received from a remote peer that currently owns
	// input). Only MouseMoveEvent, MouseWarpEvent, MouseButtonEvent,
	// MouseScrollEvent and KeyEvent are meaningful here.
	Inject(ev Event) error
	// ScreenRect reports this node's current logical desktop bounding box
	// (design.md Decision 5: one rectangle per node, not per monitor).
	ScreenRect() (ScreenRect, error)
}

// ErrUnsupportedPlatform is returned by NewBackend when no capture/inject
// implementation exists for the running OS/session type (e.g. Wayland,
// Windows and macOS before their respective phases land — see
// tasks.md section 10).
var ErrUnsupportedPlatform = fmt.Errorf("inputshare: no input capture/injection backend available for this platform")

// newPlatformBackend is provided by exactly one platform-specific file
// (build-tag selected), following the pattern already used by
// internal/ui for per-OS code. It returns ErrUnsupportedPlatform when the
// current OS/session has no implementation yet.
var newPlatformBackend = func() (Backend, error) {
	return nil, ErrUnsupportedPlatform
}

// NewBackend constructs the capture/injection backend for the running
// platform.
func NewBackend() (Backend, error) {
	return newPlatformBackend()
}

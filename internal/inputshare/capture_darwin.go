//go:build darwin

package inputshare

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ebitengine/purego"
)

func init() {
	newPlatformBackend = newDarwinBackend
}

// darwinBackend implements Backend for macOS using CoreGraphics' Quartz
// Event Services: CGEventTapCreate for global capture, CGEventPost for
// injection, resolved via purego instead of cgo (design.md Decision 1).
//
// Unlike x11Backend, there is no separate "control connection" vs "data
// connection" split: a single CGEventTap callback both observes events
// (capture) and decides whether to let them through (Suppress/Release,
// design.md Decision 3) — there is no distinct grab primitive to call.
type darwinBackend struct {
	api *darwinAPI

	keyToHID map[byte]HIDUsage
	hidToKey map[HIDUsage]byte

	// tapCallback is compiled once (not per Start) so repeated Start/Stop
	// cycles (tasks.md 2.4) don't burn through purego.NewCallback's limited
	// callback slots, which are never released.
	tapCallbackPtr uintptr

	suppressed atomic.Bool

	// modMu/modifierHeld track modifier press/release state ourselves:
	// macOS reports modifier keys via kCGEventFlagsChanged, which carries
	// no pressed/released flag of its own (see dispatchFlagsChanged) — only
	// "this modifier's state just changed" — so the backend has to
	// remember which modifiers it last considered held.
	modMu        sync.Mutex
	modifierHeld map[byte]bool

	// runMu guards the fields below, all only meaningful between a
	// successful Start and the matching Stop.
	runMu   sync.Mutex
	cb      Callbacks
	tap     uintptr // CFMachPortRef
	source  uintptr // CFRunLoopSourceRef
	runLoop uintptr // CFRunLoopRef of the dedicated capture goroutine
	stopped chan struct{}
}

func newDarwinBackend() (Backend, error) {
	api, err := loadDarwinAPI()
	if err != nil {
		return nil, fmt.Errorf("inputshare/darwin: %w", err)
	}
	return &darwinBackend{
		api:          api,
		keyToHID:     buildKeycodeToHIDMap(),
		hidToKey:     buildHIDToKeycodeMap(),
		modifierHeld: make(map[byte]bool),
	}, nil
}

func (b *darwinBackend) Name() string { return "darwin" }

func (b *darwinBackend) ScreenRect() (ScreenRect, error) {
	did := b.api.CGMainDisplayID()
	w := b.api.CGDisplayPixelsWide(did)
	h := b.api.CGDisplayPixelsHigh(did)
	return ScreenRect{WidthPx: int(w), HeightPx: int(h)}, nil
}

// missingAccessibilityMsg and missingInputMonitoringMsg name both TCC-gated
// permissions this backend needs, and where to grant them, since macOS
// gives no clean way to distinguish which one specifically blocked
// CGEventTapCreate (design.md Decision 5, tasks.md 6.1/6.2).
const permissionOnboardingMsg = "OmniDesk precisa das permissões de Acessibilidade e Monitoramento de Entrada para compartilhar mouse/teclado neste Mac. Conceda-as em Ajustes do Sistema > Privacidade e Segurança > Acessibilidade, e também em Privacidade e Segurança > Monitoramento de Entrada, depois reinicie o OmniDesk."

// Start checks the Accessibility permission (prompting for it if not yet
// granted), then creates the event tap and starts a dedicated OS thread
// running its CFRunLoop (design.md Decision 1: purego requires driving the
// run loop explicitly, since there's no C runtime doing it implicitly).
func (b *darwinBackend) Start(ctx context.Context, cb Callbacks) error {
	if !b.checkAccessibility() {
		return fmt.Errorf("inputshare/darwin: permissão de Acessibilidade não concedida. %s", permissionOnboardingMsg)
	}

	if b.tapCallbackPtr == 0 {
		b.tapCallbackPtr = purego.NewCallback(b.tapCallback)
	}

	b.runMu.Lock()
	b.cb = cb
	b.stopped = make(chan struct{})
	b.runMu.Unlock()

	startErr := make(chan error, 1)
	go b.runLoopThread(startErr)

	if err := <-startErr; err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		b.Stop()
	}()

	return nil
}

// checkAccessibility calls AXIsProcessTrustedWithOptions with the "prompt"
// option set, which both checks the permission and — if it isn't granted
// yet — triggers the system's own consent dialog (design.md Decision 5).
func (b *darwinBackend) checkAccessibility() bool {
	dict := b.api.axOptionsWithPrompt()
	defer b.api.CFRelease(dict)
	return b.api.AXIsProcessTrustedWithOptions(dict)
}

// runLoopThread is the dedicated OS thread (runtime.LockOSThread) that owns
// the event tap's CFRunLoop for as long as capture is active. It reports
// whether startup succeeded on startErr exactly once, then — only on
// success — blocks in CFRunLoopRun until Stop calls CFRunLoopStop on the
// same run loop from another goroutine.
func (b *darwinBackend) runLoopThread(startErr chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tap := b.api.CGEventTapCreate(
		cgSessionEventTap, cgHeadInsertEventTap, cgEventTapOptionDefault,
		darwinEventMask, b.tapCallbackPtr, 0,
	)
	if tap == 0 {
		// Accessibility is already confirmed granted at this point (Start
		// checked it), so a NULL tap here is the practical signal that
		// Input Monitoring is also required (design.md Decision 5: there is
		// no public API to check Input Monitoring directly).
		startErr <- fmt.Errorf("inputshare/darwin: CGEventTapCreate falhou mesmo com Acessibilidade concedida — provavelmente falta Monitoramento de Entrada. %s", permissionOnboardingMsg)
		close(b.stopped)
		return
	}

	source := b.api.CFMachPortCreateRunLoopSource(0, tap, 0)
	rl := b.api.CFRunLoopGetCurrent()
	b.api.CFRunLoopAddSource(rl, source, b.api.kCFRunLoopCommonModes)
	b.api.CGEventTapEnable(tap, true)

	b.runMu.Lock()
	b.tap = tap
	b.source = source
	b.runLoop = rl
	b.runMu.Unlock()

	startErr <- nil

	b.api.CFRunLoopRun() // blocks until Stop calls CFRunLoopStop(rl)

	b.api.CGEventTapEnable(tap, false)
	b.api.CFRelease(source)
	b.api.CFRelease(tap)

	b.runMu.Lock()
	b.tap, b.source, b.runLoop = 0, 0, 0
	b.runMu.Unlock()

	close(b.stopped)
}

// Stop asks the capture goroutine's run loop to return and waits for it to
// finish tearing down, mirroring x11Backend.Stop's "wait for the loop to
// actually end" contract. Safe to call multiple times or when Start never
// succeeded.
func (b *darwinBackend) Stop() {
	b.runMu.Lock()
	rl := b.runLoop
	stopped := b.stopped
	b.runMu.Unlock()

	if rl != 0 {
		b.api.CFRunLoopStop(rl)
	}
	if stopped != nil {
		<-stopped
	}
}

// tapCallback runs on the capture goroutine's OS thread, synchronously,
// for every event matching darwinEventMask (design.md Risk 1: it must never
// block — no I/O, no lock contention beyond the cheap mutexes below).
func (b *darwinBackend) tapCallback(proxy uintptr, eventType uint32, event uintptr, userInfo uintptr) uintptr {
	switch eventType {
	case cgEventTapDisabledByTimeout, cgEventTapDisabledByUserInput:
		// The OS disables a tap that misses its callback budget (design.md
		// Risk 2 — the macOS analog of the RECORD desync this project hit
		// for real on Linux). Re-enable immediately rather than leaving
		// capture silently dead.
		log.Printf("[inputshare/darwin] tap desabilitado pelo sistema (motivo=0x%X), reabilitando", eventType)
		b.runMu.Lock()
		tap := b.tap
		b.runMu.Unlock()
		if tap != 0 {
			b.api.CGEventTapEnable(tap, true)
		}
		return event

	case cgEventMouseMoved, cgEventLeftMouseDragged, cgEventRightMouseDragged, cgEventOtherMouseDragged:
		b.dispatchMotion(event)
	case cgEventLeftMouseDown:
		b.dispatchButton(MouseButtonLeft, true)
	case cgEventLeftMouseUp:
		b.dispatchButton(MouseButtonLeft, false)
	case cgEventRightMouseDown:
		b.dispatchButton(MouseButtonRight, true)
	case cgEventRightMouseUp:
		b.dispatchButton(MouseButtonRight, false)
	case cgEventOtherMouseDown:
		b.dispatchButton(MouseButtonMiddle, true)
	case cgEventOtherMouseUp:
		b.dispatchButton(MouseButtonMiddle, false)
	case cgEventScrollWheel:
		b.dispatchScroll(event)
	case cgEventKeyDown:
		b.dispatchKey(event, true)
	case cgEventKeyUp:
		b.dispatchKey(event, false)
	case cgEventFlagsChanged:
		b.dispatchFlagsChanged(event)
	}

	// Suppress/Release (design.md Decision 3): capture (the dispatch calls
	// above) always happens; only local *delivery* is gated by returning
	// NULL instead of the original event.
	if b.suppressed.Load() {
		return 0
	}
	return event
}

// dispatchMotion reports the cursor's clamped absolute position (loc, from
// CGEventGetLocation — used for edge detection, where "at the screen edge"
// is exactly what we want) alongside the *raw* hardware motion delta
// (kCGMouseEventDeltaX/Y) rather than a delta computed from consecutive
// absolute positions.
//
// Those two diverge exactly at a screen edge, which is exactly where this
// backend's forwarded deltas matter most: once the OS clamps the real
// cursor there, consecutive CGEventGetLocation reads stop changing even as
// the user keeps physically moving the trackpad/mouse in that direction,
// so an absolute-position-diff delta silently drops to zero right as a
// session starts sending — the remote cursor gets stuck a few pixels past
// the entry point and never travels the rest of the destination screen.
// kCGMouseEventDeltaX/Y report the actual unclamped hardware motion for
// this specific event regardless of where the OS pinned the cursor, which
// is what a "continue pushing past the edge" gesture needs forwarded.
func (b *darwinBackend) dispatchMotion(event uintptr) {
	loc := b.api.CGEventGetLocation(event)
	absX, absY := int(loc.X), int(loc.Y)

	dx := b.api.CGEventGetIntegerValueField(event, cgMouseEventDeltaX)
	dy := b.api.CGEventGetIntegerValueField(event, cgMouseEventDeltaY)

	b.runMu.Lock()
	cb := b.cb
	b.runMu.Unlock()
	if cb.OnMotion != nil {
		cb.OnMotion(absX, absY, clampDelta16(int32(dx)), clampDelta16(int32(dy)))
	}
}

func (b *darwinBackend) dispatchButton(btn MouseButton, pressed bool) {
	b.runMu.Lock()
	cb := b.cb
	b.runMu.Unlock()
	if cb.OnButton != nil {
		cb.OnButton(btn, pressed)
	}
}

// dispatchScroll reads both wheel axes directly off the event
// (kCGScrollWheelEventDeltaAxis1/2), matching MouseScrollEvent's own
// "positive = down/right" convention (event.go).
func (b *darwinBackend) dispatchScroll(event uintptr) {
	axis1 := b.api.CGEventGetIntegerValueField(event, cgScrollWheelEventDeltaAxis1)
	axis2 := b.api.CGEventGetIntegerValueField(event, cgScrollWheelEventDeltaAxis2)

	b.runMu.Lock()
	cb := b.cb
	b.runMu.Unlock()
	if cb.OnScroll != nil {
		// axis1/axis2 are positive for an upward/leftward wheel motion —
		// the opposite sign of the wire protocol's "positive = down/right".
		cb.OnScroll(clampDelta16(int32(-axis2)), clampDelta16(int32(-axis1)))
	}
}

func (b *darwinBackend) dispatchKey(event uintptr, pressed bool) {
	vk := byte(b.api.CGEventGetIntegerValueField(event, cgKeyboardEventKeycode))
	hid, ok := b.keyToHID[vk]
	if !ok {
		log.Printf("[inputshare/darwin] keycode %d sem mapeamento HID conhecido, ignorado", vk)
		return
	}
	b.runMu.Lock()
	cb := b.cb
	b.runMu.Unlock()
	if cb.OnKey != nil {
		cb.OnKey(hid, pressed)
	}
}

// dispatchFlagsChanged handles modifier keys, which macOS reports via
// kCGEventFlagsChanged instead of KeyDown/KeyUp. The event's keycode field
// still names the specific modifier that changed (e.g. left vs right
// Shift), but the event carries no pressed/released bit of its own — every
// FlagsChanged for a given key toggles that key's held state, so the
// backend tracks it itself.
func (b *darwinBackend) dispatchFlagsChanged(event uintptr) {
	vk := byte(b.api.CGEventGetIntegerValueField(event, cgKeyboardEventKeycode))
	hid, ok := b.keyToHID[vk]
	if !ok {
		log.Printf("[inputshare/darwin] keycode de modificador %d sem mapeamento HID conhecido, ignorado", vk)
		return
	}

	b.modMu.Lock()
	held := !b.modifierHeld[vk]
	b.modifierHeld[vk] = held
	b.modMu.Unlock()

	b.runMu.Lock()
	cb := b.cb
	b.runMu.Unlock()
	if cb.OnKey != nil {
		cb.OnKey(hid, held)
	}
}

func clampDelta16(d int32) int16 {
	const max = 32767
	if d > max {
		return max
	}
	if d < -max {
		return -max
	}
	return int16(d)
}

// currentLocation returns the live global cursor position. CGEventCreate(NULL)
// followed by CGEventGetLocation is the standard Quartz technique for
// reading current input state without an existing event in hand — needed
// because, unlike XTest's relative FakeInput mode, CGEventCreateMouseEvent
// only accepts an absolute CGPoint (there is no native relative-motion
// event on macOS).
func (b *darwinBackend) currentLocation() (CGPoint, error) {
	ev := b.api.CGEventCreate(0)
	if ev == 0 {
		return CGPoint{}, fmt.Errorf("inputshare/darwin: CGEventCreate(NULL) failed")
	}
	defer b.api.CFRelease(ev)
	return b.api.CGEventGetLocation(ev), nil
}

func (b *darwinBackend) postMouseEvent(eventType uint32, pt CGPoint, button uint32) error {
	ev := b.api.CGEventCreateMouseEvent(0, eventType, pt, button)
	if ev == 0 {
		return fmt.Errorf("inputshare/darwin: CGEventCreateMouseEvent failed (type=%d)", eventType)
	}
	defer b.api.CFRelease(ev)
	b.api.CGEventPost(cgHIDEventTap, ev)
	return nil
}

func (b *darwinBackend) Inject(ev Event) error {
	switch e := ev.(type) {
	case MouseMoveEvent:
		cur, err := b.currentLocation()
		if err != nil {
			return err
		}
		pt := CGPoint{X: cur.X + float64(e.DX), Y: cur.Y + float64(e.DY)}
		return b.postMouseEvent(cgEventMouseMoved, pt, 0)

	case MouseWarpEvent:
		return b.postMouseEvent(cgEventMouseMoved, CGPoint{X: float64(e.X), Y: float64(e.Y)}, 0)

	case MouseButtonEvent:
		cur, err := b.currentLocation()
		if err != nil {
			return err
		}
		eventType, cgButton := darwinButtonEvent(e.Button, e.Pressed)
		return b.postMouseEvent(eventType, cur, cgButton)

	case MouseScrollEvent:
		return b.injectScroll(e)

	case KeyEvent:
		vk, ok := b.hidToKey[e.HID]
		if !ok {
			return fmt.Errorf("inputshare/darwin: no native keycode mapped for HID 0x%02X", e.HID)
		}
		kev := b.api.CGEventCreateKeyboardEvent(0, uint16(vk), e.Pressed)
		if kev == 0 {
			return fmt.Errorf("inputshare/darwin: CGEventCreateKeyboardEvent failed for VK %d", vk)
		}
		defer b.api.CFRelease(kev)
		b.api.CGEventPost(cgHIDEventTap, kev)
		return nil

	default:
		return nil
	}
}

// darwinButtonEvent maps our platform-independent MouseButton/pressed pair
// to CoreGraphics' (CGEventType, CGMouseButton) pair.
func darwinButtonEvent(btn MouseButton, pressed bool) (eventType uint32, cgButton uint32) {
	switch btn {
	case MouseButtonRight:
		cgButton = 1
		if pressed {
			return cgEventRightMouseDown, cgButton
		}
		return cgEventRightMouseUp, cgButton
	case MouseButtonMiddle:
		cgButton = 2
		if pressed {
			return cgEventOtherMouseDown, cgButton
		}
		return cgEventOtherMouseUp, cgButton
	default: // MouseButtonLeft and anything unrecognized
		cgButton = 0
		if pressed {
			return cgEventLeftMouseDown, cgButton
		}
		return cgEventLeftMouseUp, cgButton
	}
}

// injectScroll builds a scroll-wheel CGEvent by hand (CGEventCreate +
// CGEventSetType + CGEventSetIntegerValueField) instead of calling the
// variadic CGEventCreateScrollWheelEvent. That function's C prototype takes
// a variable number of wheel-delta arguments, and Apple's arm64 calling
// convention requires ALL arguments to a variadic call — not just the
// variadic tail — to be passed on the stack; purego's RegisterLibFunc has
// no way to know a symbol is variadic and places every argument in
// registers as a normal fixed-arity call would. Verified directly on Apple
// Silicon while implementing this: a single-wheel call happened to read
// back correctly, but the two-wheel form (needed for horizontal scroll)
// silently corrupted the second delta. Setting the fields on a manually
// created event has no such ambiguity and produces the identical CGEvent.
func (b *darwinBackend) injectScroll(e MouseScrollEvent) error {
	ev := b.api.CGEventCreate(0)
	if ev == 0 {
		return fmt.Errorf("inputshare/darwin: CGEventCreate(NULL) failed for scroll")
	}
	defer b.api.CFRelease(ev)
	b.api.CGEventSetType(ev, cgEventScrollWheel)
	// Wire convention is "positive = down/right" (event.go); Quartz's axes
	// are positive for up/left, so negate both (see dispatchScroll).
	b.api.CGEventSetIntegerValueField(ev, cgScrollWheelEventDeltaAxis1, int64(-e.DY))
	b.api.CGEventSetIntegerValueField(ev, cgScrollWheelEventDeltaAxis2, int64(-e.DX))
	b.api.CGEventPost(cgHIDEventTap, ev)
	return nil
}

// Suppress/Release just flip the flag tapCallback checks on every event —
// no CGEventTap re-creation, no extra round-trip (design.md Decision 3).
func (b *darwinBackend) Suppress() error {
	b.suppressed.Store(true)
	return nil
}

func (b *darwinBackend) Release() error {
	b.suppressed.Store(false)
	return nil
}

const cgHIDEventTap = 0

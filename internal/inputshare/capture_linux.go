//go:build linux

package inputshare

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/record"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgb/xtest"
)

func init() {
	newPlatformBackend = newX11Backend
}

// x11Backend implements Backend for Linux/X11 using the pure-Go xgb
// bindings (no cgo, keeping the project's existing convention — see
// internal/ui/systray_darwin_nocgo.go): XTest for injection and the RECORD
// extension for global capture.
//
// The connection is split in two (tasks.md 7.6, verified against a running
// X server): this xgb connection is the *control* connection, used for
// XTest injection and for RECORD's Create/DisableContext, while the
// streaming EnableContext request lives on a separate hand-rolled
// connection (recordConn, record_conn_linux.go). That split is mandatory,
// not stylistic — the X server keeps emitting further EnableContext reply
// packets under the original request's sequence number, which xgb's
// one-reply-per-cookie dispatch cannot represent. Measured on Xorg with
// both halves on one xgb connection: capture stopped after the initial
// StartOfData chunk and the very first checked FakeInput afterwards
// blocked forever.
type x11Backend struct {
	// conn is the control connection: XTest injection plus RECORD's
	// Create/DisableContext. rec is the data connection, carrying only
	// the EnableContext stream.
	conn *xgb.Conn
	rec  *recordConn
	root xproto.Window

	screenW, screenH int
	ctxID            record.Context

	keyToHID map[byte]HIDUsage
	hidToKey map[HIDUsage]byte

	mu         sync.Mutex
	lastX      int32
	lastY      int32
	haveLastXY bool

	cb      Callbacks
	stopped chan struct{}
}

func newX11Backend() (Backend, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		// No X server reachable — most commonly a Wayland-only session
		// (no XWayland) or a headless machine. Treat exactly like any
		// other unsupported platform (task 7.5).
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedPlatform, err)
	}

	if err := xtest.Init(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("inputshare/x11: XTest extension unavailable: %w", err)
	}
	if err := record.Init(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("inputshare/x11: RECORD extension unavailable: %w", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)

	ctxID, err := record.NewContextId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("inputshare/x11: failed to allocate RECORD context id: %w", err)
	}

	return &x11Backend{
		conn:     conn,
		root:     screen.Root,
		screenW:  int(screen.WidthInPixels),
		screenH:  int(screen.HeightInPixels),
		ctxID:    ctxID,
		keyToHID: buildKeycodeToHIDMap(),
		hidToKey: buildHIDToKeycodeMap(),
	}, nil
}

func (b *x11Backend) Name() string { return "x11" }

func (b *x11Backend) ScreenRect() (ScreenRect, error) {
	return ScreenRect{WidthPx: b.screenW, HeightPx: b.screenH}, nil
}

// recordDeviceEvents covers the core protocol event codes we care about:
// KeyPress(2), KeyRelease(3), ButtonPress(4), ButtonRelease(5),
// MotionNotify(6).
var recordDeviceEvents = record.Range8{First: xproto.KeyPress, Last: xproto.MotionNotify}

func (b *x11Backend) Start(ctx context.Context, cb Callbacks) error {
	b.cb = cb
	b.stopped = make(chan struct{})

	ranges := []record.Range{{DeviceEvents: recordDeviceEvents}}
	clientSpecs := []record.ClientSpec{record.CsAllClients}

	if err := record.CreateContextChecked(
		b.conn, b.ctxID, record.ElementHeader(0),
		uint32(len(clientSpecs)), uint32(len(ranges)),
		clientSpecs, ranges,
	).Check(); err != nil {
		close(b.stopped)
		return fmt.Errorf("inputshare/x11: RECORD CreateContext failed: %w", err)
	}

	rec, err := dialRecordConn()
	if err != nil {
		record.FreeContext(b.conn, b.ctxID)
		close(b.stopped)
		return fmt.Errorf("inputshare/x11: RECORD data connection failed: %w", err)
	}
	if err := rec.enableContext(uint32(b.ctxID)); err != nil {
		rec.Close()
		record.FreeContext(b.conn, b.ctxID)
		close(b.stopped)
		return err
	}
	b.rec = rec

	go b.recordLoop()

	go func() {
		<-ctx.Done()
		b.Stop()
	}()

	return nil
}

func (b *x11Backend) recordLoop() {
	defer close(b.stopped)
	for {
		chunk, err := b.rec.next()
		if err != nil {
			log.Printf("[inputshare/x11] RECORD stream ended: %v", err)
			return
		}
		// Category 0 = FromServer: raw device events, which is the only
		// category we asked for real event bytes on. Other categories
		// (StartOfData/EndOfData/ClientStarted/ClientDied) carry no core
		// protocol events and are safely ignored.
		if chunk.Category != 0 || len(chunk.Data) == 0 {
			continue
		}
		b.parseChunk(chunk.Data)
	}
}

// parseChunk splits a RECORD data chunk into the standard, fixed 32-byte
// core X protocol events it is defined to contain and dispatches each one.
func (b *x11Backend) parseChunk(data []byte) {
	const eventSize = 32
	for len(data) >= eventSize {
		raw := data[:eventSize]
		data = data[eventSize:]

		// High bit marks "sent via SendEvent"; irrelevant for real device
		// input, and RECORD-delivered core events don't set it, but strip
		// it defensively before dispatch.
		code := raw[0] & 0x7f

		switch code {
		case xproto.KeyPress, xproto.KeyRelease:
			b.dispatchKey(raw, code == xproto.KeyPress)
		case xproto.ButtonPress, xproto.ButtonRelease:
			b.dispatchButton(raw, code == xproto.ButtonPress)
		case xproto.MotionNotify:
			b.dispatchMotion(raw)
		}
	}
}

func (b *x11Backend) dispatchKey(raw []byte, pressed bool) {
	ev := xproto.KeyPressEventNew(raw).(xproto.KeyPressEvent)
	hid, ok := b.keyToHID[byte(ev.Detail)]
	if !ok {
		log.Printf("[inputshare/x11] keycode %d sem mapeamento HID conhecido, ignorado", ev.Detail)
		return
	}
	if b.cb.OnKey != nil {
		b.cb.OnKey(hid, pressed)
	}
}

func (b *x11Backend) dispatchButton(raw []byte, pressed bool) {
	ev := xproto.ButtonPressEventNew(raw).(xproto.ButtonPressEvent)
	switch ev.Detail {
	case 1:
		if b.cb.OnButton != nil {
			b.cb.OnButton(MouseButtonLeft, pressed)
		}
	case 2:
		if b.cb.OnButton != nil {
			b.cb.OnButton(MouseButtonMiddle, pressed)
		}
	case 3:
		if b.cb.OnButton != nil {
			b.cb.OnButton(MouseButtonRight, pressed)
		}
	case 4: // scroll up
		if pressed && b.cb.OnScroll != nil {
			b.cb.OnScroll(0, -1)
		}
	case 5: // scroll down
		if pressed && b.cb.OnScroll != nil {
			b.cb.OnScroll(0, 1)
		}
	case 6: // scroll left
		if pressed && b.cb.OnScroll != nil {
			b.cb.OnScroll(-1, 0)
		}
	case 7: // scroll right
		if pressed && b.cb.OnScroll != nil {
			b.cb.OnScroll(1, 0)
		}
	}
}

func (b *x11Backend) dispatchMotion(raw []byte) {
	ev := xproto.MotionNotifyEventNew(raw).(xproto.MotionNotifyEvent)

	b.mu.Lock()
	var dx, dy int32
	if b.haveLastXY {
		dx = int32(ev.RootX) - b.lastX
		dy = int32(ev.RootY) - b.lastY
	}
	b.lastX, b.lastY = int32(ev.RootX), int32(ev.RootY)
	b.haveLastXY = true
	b.mu.Unlock()

	if b.cb.OnMotion != nil {
		b.cb.OnMotion(int(ev.RootX), int(ev.RootY), clampDelta(dx), clampDelta(dy))
	}
}

func clampDelta(d int32) int16 {
	const max = 32767
	if d > max {
		return max
	}
	if d < -max {
		return -max
	}
	return int16(d)
}

func (b *x11Backend) Stop() {
	// Tell the server to stop recording, then drop the data connection.
	// Closing it is what actually unblocks recordLoop: its read returns an
	// error immediately, so Stop never depends on the server choosing to
	// send one more packet.
	record.DisableContext(b.conn, b.ctxID)
	if b.rec != nil {
		b.rec.Close()
	}
	// The control connection is intentionally left open until the process
	// exits; closing it here could race with in-flight FakeInput calls from
	// Manager.StopSession's synchronous release-all path.
	if b.stopped != nil {
		<-b.stopped
	}
}

// x11ButtonDetail maps our platform-independent MouseButton to the X11
// button number convention (2 = middle, 3 = right — swapped relative to
// our own Right=2/Middle=3 ordering).
func x11ButtonDetail(btn MouseButton) byte {
	switch btn {
	case MouseButtonLeft:
		return 1
	case MouseButtonMiddle:
		return 2
	case MouseButtonRight:
		return 3
	default:
		return 1
	}
}

func (b *x11Backend) Inject(ev Event) error {
	switch e := ev.(type) {
	case MouseMoveEvent:
		// Unchecked (fire-and-forget): FakeInputChecked(...).Check() forces
		// a full round trip to the X server (xgb's Cookie.Check calls
		// conn.Sync() when nothing has piggy-backed a reply yet — see
		// cookie.go). Inject runs synchronously inside Conn.readLoop, so
		// that round trip blocks reading the *next* websocket frame too;
		// at real mouse/trackpad event rates this serializes into visible
		// stutter on the controlled machine. Motion is exactly the case
		// where losing an occasional sample silently is fine and a stuck
		// connection is not — unlike a key or click, which still use the
		// checked/blocking form below.
		xtest.FakeInput(b.conn, xproto.MotionNotify, 1, xproto.TimeCurrentTime, b.root, e.DX, e.DY, 0)
		return nil

	case MouseWarpEvent:
		xtest.FakeInput(b.conn, xproto.MotionNotify, 0, xproto.TimeCurrentTime, b.root, int16(e.X), int16(e.Y), 0)
		return nil

	case MouseButtonEvent:
		t := byte(xproto.ButtonRelease)
		if e.Pressed {
			t = xproto.ButtonPress
		}
		return xtest.FakeInputChecked(b.conn, t, x11ButtonDetail(e.Button), xproto.TimeCurrentTime, b.root, 0, 0, 0).Check()

	case MouseScrollEvent:
		return b.injectScroll(e)

	case KeyEvent:
		keycode, ok := b.hidToKey[e.HID]
		if !ok {
			return fmt.Errorf("inputshare/x11: no native keycode mapped for HID 0x%02X", e.HID)
		}
		t := byte(xproto.KeyRelease)
		if e.Pressed {
			t = xproto.KeyPress
		}
		return xtest.FakeInputChecked(b.conn, t, keycode, xproto.TimeCurrentTime, b.root, 0, 0, 0).Check()

	default:
		return nil
	}
}

// injectScroll synthesizes wheel button click pairs, one per unit of
// scroll delta (X11 has no continuous wheel event — see design.md's
// discussion of scroll handling).
func (b *x11Backend) injectScroll(e MouseScrollEvent) error {
	const maxTicks = 20 // guard against a pathological delta flooding the connection

	click := func(detail byte) error {
		if err := xtest.FakeInputChecked(b.conn, xproto.ButtonPress, detail, xproto.TimeCurrentTime, b.root, 0, 0, 0).Check(); err != nil {
			return err
		}
		return xtest.FakeInputChecked(b.conn, xproto.ButtonRelease, detail, xproto.TimeCurrentTime, b.root, 0, 0, 0).Check()
	}

	if e.DY != 0 {
		detail := byte(4)
		n := int(e.DY)
		if n < 0 {
			n = -n
		} else {
			detail = 5
		}
		if n > maxTicks {
			n = maxTicks
		}
		for i := 0; i < n; i++ {
			if err := click(detail); err != nil {
				return err
			}
		}
	}
	if e.DX != 0 {
		detail := byte(6)
		n := int(e.DX)
		if n < 0 {
			n = -n
		} else {
			detail = 7
		}
		if n > maxTicks {
			n = maxTicks
		}
		for i := 0; i < n; i++ {
			if err := click(detail); err != nil {
				return err
			}
		}
	}
	return nil
}

// Suppress grabs the pointer and keyboard on the root window with
// owner-events disabled and an empty event mask: the X server stops
// delivering button, motion and key events to every other client (this
// machine's own desktop included) for as long as the grab holds, while our
// RECORD stream — a passive spy, unaffected by grabs — keeps seeing
// everything to forward on. GrabModeAsync for both devices keeps normal
// event processing/timing intact; only delivery is redirected away from
// local apps.
func (b *x11Backend) Suppress() error {
	ptr, err := xproto.GrabPointer(
		b.conn, false, b.root, 0,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
		0, 0, xproto.TimeCurrentTime,
	).Reply()
	if err != nil {
		return fmt.Errorf("inputshare/x11: GrabPointer failed: %w", err)
	}
	if ptr.Status != xproto.GrabStatusSuccess {
		return fmt.Errorf("inputshare/x11: GrabPointer refused (status %d)", ptr.Status)
	}

	kbd, err := xproto.GrabKeyboard(
		b.conn, false, b.root, xproto.TimeCurrentTime,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
	).Reply()
	if err != nil {
		xproto.UngrabPointer(b.conn, xproto.TimeCurrentTime)
		return fmt.Errorf("inputshare/x11: GrabKeyboard failed: %w", err)
	}
	if kbd.Status != xproto.GrabStatusSuccess {
		xproto.UngrabPointer(b.conn, xproto.TimeCurrentTime)
		return fmt.Errorf("inputshare/x11: GrabKeyboard refused (status %d)", kbd.Status)
	}

	return nil
}

// Release undoes Suppress. Both requests are unchecked and best-effort: by
// the time a session is ending (peer gone, network dropped) the connection
// itself may already be in a bad state, and there is nothing more useful to
// do with an ungrab error than log noise — the grab is also implicitly
// dropped if this process's X connection ever closes.
func (b *x11Backend) Release() error {
	xproto.UngrabPointer(b.conn, xproto.TimeCurrentTime)
	xproto.UngrabKeyboard(b.conn, xproto.TimeCurrentTime)
	return nil
}

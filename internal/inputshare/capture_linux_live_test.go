//go:build linux

package inputshare

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BurntSushi/xgb/xproto"
)

// TestX11LiveCaptureAndInject exercises the real X11 backend against the X
// server named by DISPLAY: it starts RECORD capture, injects pointer motion
// through XTest, and checks that the injected events come back out of the
// capture stream.
//
// It is the regression test for tasks.md 7.6. Two failures it is meant to
// catch, both of which happened when EnableContext shared the ordinary xgb
// connection: capture that stops after the first chunk (motions stays at 0
// or 1), and injection that blocks forever once the RECORD stream has
// desynchronised the connection's reply bookkeeping.
//
// It moves the real cursor, so it is opt-in:
//
//	OMNIDESK_X11_LIVE_TEST=1 go test ./internal/inputshare -run TestX11Live -v
func TestX11LiveCaptureAndInject(t *testing.T) {
	if os.Getenv("OMNIDESK_X11_LIVE_TEST") == "" {
		t.Skip("set OMNIDESK_X11_LIVE_TEST=1 to run against the local X server (moves the cursor)")
	}

	backend, err := newX11Backend()
	if err != nil {
		t.Fatalf("newX11Backend: %v", err)
	}
	b := backend.(*x11Backend)

	origin, err := xproto.QueryPointer(b.conn, b.root).Reply()
	if err != nil {
		t.Fatalf("QueryPointer: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Inject(MouseWarpEvent{X: uint16(origin.RootX), Y: uint16(origin.RootY)})
	})

	var motions int64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Start(ctx, Callbacks{
		OnMotion: func(absX, absY int, dx, dy int16) { atomic.AddInt64(&motions, 1) },
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Give the RECORD context time to reach the enabled state before the
	// first injection, otherwise early events are legitimately missed.
	time.Sleep(300 * time.Millisecond)

	const injections = 20
	for i := 0; i < injections; i++ {
		// A tight box around wherever the cursor already is, so a developer
		// running this locally barely notices.
		x := uint16(int(origin.RootX) + i%5)
		y := uint16(int(origin.RootY) + (i/5)%5)

		done := make(chan error, 1)
		go func() { done <- b.Inject(MouseWarpEvent{X: x, Y: y}) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Inject %d: %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Inject %d blocked for 2s: the control connection's reply "+
				"bookkeeping is desynchronised (see record_conn_linux.go)", i)
		}
		time.Sleep(30 * time.Millisecond)
	}

	// Let the tail of the stream drain.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt64(&motions) < injections {
		time.Sleep(50 * time.Millisecond)
	}

	got := atomic.LoadInt64(&motions)
	t.Logf("injected %d motion events, captured %d", injections, got)
	switch {
	case got == 0:
		t.Fatalf("captured no motion events at all: the RECORD stream delivered nothing")
	case got < injections/2:
		t.Fatalf("captured only %d of %d motion events: the RECORD stream stopped early", got, injections)
	}

	stopped := make(chan struct{})
	go func() { b.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return within 3s: the record loop is still blocked")
	}
}

// TestX11LiveKeyAndButtonRoundTrip covers what TestX11LiveCaptureAndInject
// deliberately leaves out: keyboard and mouse-button events. It injects a
// representative sample of keymap_linux.go's evdevToHID table (letters,
// digits, punctuation, modifiers) plus a mouse click, and asserts every one
// comes back through the capture stream with the same HID usage and
// pressed/released state, in order.
//
// This only proves the injection→RECORD→dispatch plumbing is sound for
// these event types (the same thing TestX11LiveCaptureAndInject proved for
// motion) — it round-trips through keymap_linux.go's own table in both
// directions, so it cannot catch a HID usage mistakenly paired with the
// wrong evdev code in that table (inject and capture would agree on the
// same wrong key). Pair it with a quick manual check: run this on a plain
// text field with your intended keyboard layout and confirm what actually
// gets typed matches what the log below claims was injected.
//
//	OMNIDESK_X11_LIVE_TEST=1 go test ./internal/inputshare -run TestX11LiveKeyAndButton -v
func TestX11LiveKeyAndButtonRoundTrip(t *testing.T) {
	if os.Getenv("OMNIDESK_X11_LIVE_TEST") == "" {
		t.Skip("set OMNIDESK_X11_LIVE_TEST=1 to run against the local X server (sends real key/button events)")
	}

	backend, err := newX11Backend()
	if err != nil {
		t.Fatalf("newX11Backend: %v", err)
	}
	b := backend.(*x11Backend)

	keys := []HIDUsage{
		HIDKeyA, HIDKeyZ, HIDKeyM,
		HIDKey1, HIDKey0,
		HIDKeySpace, HIDKeyEnter, HIDKeyTab, HIDKeyBackspace,
		HIDKeyComma, HIDKeyPeriod, HIDKeySlash, HIDKeyMinus, HIDKeyEqual,
		HIDKeyLeftShift, HIDKeyLeftControl, HIDKeyLeftAlt,
		HIDKeyUp, HIDKeyDown, HIDKeyLeft, HIDKeyRight,
	}

	type gotEvent struct {
		key     *KeyEvent
		btn     *MouseButtonEvent
		scrollY int16
	}
	var (
		mu  sync.Mutex
		got []gotEvent
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Start(ctx, Callbacks{
		OnKey: func(hid HIDUsage, pressed bool) {
			mu.Lock()
			got = append(got, gotEvent{key: &KeyEvent{HID: hid, Pressed: pressed}})
			mu.Unlock()
		},
		OnButton: func(btn MouseButton, pressed bool) {
			mu.Lock()
			got = append(got, gotEvent{btn: &MouseButtonEvent{Button: btn, Pressed: pressed}})
			mu.Unlock()
		},
		OnScroll: func(dx, dy int16) {
			mu.Lock()
			got = append(got, gotEvent{scrollY: dy})
			mu.Unlock()
		},
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(b.Stop)

	time.Sleep(300 * time.Millisecond)

	inject := func(ev Event) {
		done := make(chan error, 1)
		go func() { done <- b.Inject(ev) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Inject(%#v): %v", ev, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Inject(%#v) blocked for 2s", ev)
		}
		time.Sleep(20 * time.Millisecond)
	}

	var want []gotEvent
	for _, k := range keys {
		inject(KeyEvent{HID: k, Pressed: true})
		inject(KeyEvent{HID: k, Pressed: false})
		want = append(want,
			gotEvent{key: &KeyEvent{HID: k, Pressed: true}},
			gotEvent{key: &KeyEvent{HID: k, Pressed: false}},
		)
	}
	inject(MouseButtonEvent{Button: MouseButtonLeft, Pressed: true})
	inject(MouseButtonEvent{Button: MouseButtonLeft, Pressed: false})
	want = append(want,
		gotEvent{btn: &MouseButtonEvent{Button: MouseButtonLeft, Pressed: true}},
		gotEvent{btn: &MouseButtonEvent{Button: MouseButtonLeft, Pressed: false}},
	)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= len(want) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("injected %d key/button events, captured %d", len(want), len(got))
	if len(got) < len(want) {
		t.Fatalf("captured only %d of %d key/button events — check for \"sem mapeamento HID conhecido\" warnings above", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		switch {
		case w.key != nil:
			if g.key == nil || *g.key != *w.key {
				t.Fatalf("event %d: want key %+v, got %+v", i, *w.key, g)
			}
		case w.btn != nil:
			if g.btn == nil || *g.btn != *w.btn {
				t.Fatalf("event %d: want button %+v, got %+v", i, *w.btn, g)
			}
		}
	}
}

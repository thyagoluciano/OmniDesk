//go:build darwin

package inputshare

import (
	"context"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDarwinSymbolResolution only dlopen's CoreFoundation and
// ApplicationServices and resolves every symbol darwinAPI needs, failing
// with a clear message naming whichever one doesn't exist (tasks.md 1.2).
// It needs no permission grant and touches no real input, so it is safe to
// run on any Mac, but it is still opt-in for consistency with every other
// live test in this package:
//
//	OMNIDESK_DARWIN_LIVE_TEST=1 go test ./internal/inputshare -run TestDarwinSymbolResolution -v
func TestDarwinSymbolResolution(t *testing.T) {
	if os.Getenv("OMNIDESK_DARWIN_LIVE_TEST") == "" {
		t.Skip("set OMNIDESK_DARWIN_LIVE_TEST=1 to run against the real system frameworks")
	}

	api, err := loadDarwinAPI()
	if err != nil {
		t.Fatalf("loadDarwinAPI: %v", err)
	}
	if api.CGEventTapCreate == nil || api.CGEventPost == nil || api.AXIsProcessTrustedWithOptions == nil {
		t.Fatal("loadDarwinAPI returned successfully but left required function pointers nil")
	}

	did := api.CGMainDisplayID()
	w := api.CGDisplayPixelsWide(did)
	h := api.CGDisplayPixelsHigh(did)
	if w <= 0 || h <= 0 {
		t.Fatalf("CGMainDisplayID/CGDisplayPixelsWide/High returned a non-positive size: %dx%d", w, h)
	}
	t.Logf("main display %d: %dx%d", did, w, h)
}

// TestDarwinStartStopLoop calls Start followed by Stop ten times on the
// same backend instance and checks that neither call hangs nor leaks a
// goroutine (tasks.md 2.4). It needs Accessibility+Input Monitoring
// already granted (same as the other live tests) but injects no events.
//
//	OMNIDESK_DARWIN_LIVE_TEST=1 go test ./internal/inputshare -run TestDarwinStartStopLoop -v
func TestDarwinStartStopLoop(t *testing.T) {
	if os.Getenv("OMNIDESK_DARWIN_LIVE_TEST") == "" {
		t.Skip("set OMNIDESK_DARWIN_LIVE_TEST=1 to run against the real system (needs Accessibility+Input Monitoring already granted)")
	}

	backend, err := newDarwinBackend()
	if err != nil {
		t.Fatalf("newDarwinBackend: %v", err)
	}
	b := backend.(*darwinBackend)

	runtime.GC()
	before := runtime.NumGoroutine()

	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- b.Start(ctx, Callbacks{}) }()
		select {
		case err := <-done:
			if err != nil {
				cancel()
				t.Fatalf("Start iteration %d: %v", i, err)
			}
		case <-time.After(3 * time.Second):
			cancel()
			t.Fatalf("Start iteration %d blocked for 3s", i)
		}

		stopped := make(chan struct{})
		go func() { b.Stop(); close(stopped) }()
		select {
		case <-stopped:
		case <-time.After(3 * time.Second):
			t.Fatalf("Stop iteration %d blocked for 3s", i)
		}
		cancel()
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	t.Logf("goroutines before=%d after=%d", before, after)
	if after > before+2 { // small slack for GC/finalizer goroutines unrelated to this backend
		t.Fatalf("goroutine count grew from %d to %d after 10 Start/Stop cycles: likely leak", before, after)
	}
}

// TestDarwinLiveCaptureAndInject starts the real darwin backend, injects
// pointer motion through CGEventPost, and checks the injected events come
// back out of the CGEventTap capture stream. It moves the real cursor and
// requires both Accessibility and Input Monitoring already granted to
// whatever process runs `go test` (there is no way to grant them
// programmatically — see permission_darwin.go's onboarding message), so it
// is opt-in:
//
//	OMNIDESK_DARWIN_LIVE_TEST=1 go test ./internal/inputshare -run TestDarwinLiveCaptureAndInject -v
func TestDarwinLiveCaptureAndInject(t *testing.T) {
	if os.Getenv("OMNIDESK_DARWIN_LIVE_TEST") == "" {
		t.Skip("set OMNIDESK_DARWIN_LIVE_TEST=1 to run against the real system (moves the cursor, needs Accessibility+Input Monitoring already granted)")
	}

	backend, err := newDarwinBackend()
	if err != nil {
		t.Fatalf("newDarwinBackend: %v", err)
	}
	b := backend.(*darwinBackend)

	origin, err := b.currentLocation()
	if err != nil {
		t.Fatalf("currentLocation: %v", err)
	}
	t.Cleanup(func() {
		_ = b.postMouseEvent(cgEventMouseMoved, origin, 0)
	})

	var motions int64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Start(ctx, Callbacks{
		OnMotion: func(absX, absY int, dx, dy int16) { atomic.AddInt64(&motions, 1) },
	}); err != nil {
		t.Fatalf("Start: %v (grant Accessibility + Input Monitoring to the process running `go test` and retry)", err)
	}
	t.Cleanup(b.Stop)

	time.Sleep(200 * time.Millisecond)

	const injections = 20
	for i := 0; i < injections; i++ {
		x := origin.X + float64(i%5)
		y := origin.Y + float64((i/5)%5)
		if err := b.postMouseEvent(cgEventMouseMoved, CGPoint{X: x, Y: y}, 0); err != nil {
			t.Fatalf("inject %d: %v", i, err)
		}
		time.Sleep(30 * time.Millisecond)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt64(&motions) < injections {
		time.Sleep(50 * time.Millisecond)
	}

	got := atomic.LoadInt64(&motions)
	t.Logf("injected %d motion events, captured %d", injections, got)
	switch {
	case got == 0:
		t.Fatal("captured no motion events at all: the CGEventTap delivered nothing")
	case got < injections/2:
		t.Fatalf("captured only %d of %d motion events: the tap stopped early", got, injections)
	}
}

// TestDarwinLiveKeyButtonScrollAndSuppress covers keyboard, mouse-button and
// scroll injection round-tripping through capture, plus Suppress/Release
// (tasks.md 3.1, 5.1, 5.2): while suppressed, an injected key must still
// reach OnKey (capture keeps working) but Inject must not be needed to
// prove local delivery is blocked — that half needs a second, real,
// external input source (osascript/cliclick) and a human watching the
// screen, which is why tasks.md 5.1/5.2 call for a manual live check in
// addition to this automated one. This test only proves the plumbing that
// CAN be automated: injection -> capture -> HID/button/scroll translation,
// and that Suppress does not stop capture from seeing the event.
//
//	OMNIDESK_DARWIN_LIVE_TEST=1 go test ./internal/inputshare -run TestDarwinLiveKeyButtonScrollAndSuppress -v
func TestDarwinLiveKeyButtonScrollAndSuppress(t *testing.T) {
	if os.Getenv("OMNIDESK_DARWIN_LIVE_TEST") == "" {
		t.Skip("set OMNIDESK_DARWIN_LIVE_TEST=1 to run against the real system (sends real key/button/scroll events, needs Accessibility+Input Monitoring already granted)")
	}

	backend, err := newDarwinBackend()
	if err != nil {
		t.Fatalf("newDarwinBackend: %v", err)
	}
	b := backend.(*darwinBackend)

	type gotEvent struct {
		key      *KeyEvent
		btn      *MouseButtonEvent
		scrollY  int16
		isScroll bool
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
			got = append(got, gotEvent{scrollY: dy, isScroll: true})
			mu.Unlock()
		},
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(b.Stop)

	time.Sleep(200 * time.Millisecond)

	// Suppress before injecting: per design.md Decision 3/tasks.md 5.2,
	// capture must keep seeing events even while local delivery is
	// suppressed.
	if err := b.Suppress(); err != nil {
		t.Fatalf("Suppress: %v", err)
	}
	t.Cleanup(func() { _ = b.Release() })

	keys := []HIDUsage{HIDKeyA, HIDKeySpace, HIDKeyLeftShift, HIDKeyEnter}
	var want []gotEvent
	for _, k := range keys {
		if err := b.Inject(KeyEvent{HID: k, Pressed: true}); err != nil {
			t.Fatalf("Inject key down %v: %v", k, err)
		}
		if err := b.Inject(KeyEvent{HID: k, Pressed: false}); err != nil {
			t.Fatalf("Inject key up %v: %v", k, err)
		}
		want = append(want,
			gotEvent{key: &KeyEvent{HID: k, Pressed: true}},
			gotEvent{key: &KeyEvent{HID: k, Pressed: false}},
		)
		time.Sleep(20 * time.Millisecond)
	}

	if err := b.Inject(MouseButtonEvent{Button: MouseButtonLeft, Pressed: true}); err != nil {
		t.Fatalf("Inject button down: %v", err)
	}
	if err := b.Inject(MouseButtonEvent{Button: MouseButtonLeft, Pressed: false}); err != nil {
		t.Fatalf("Inject button up: %v", err)
	}
	want = append(want,
		gotEvent{btn: &MouseButtonEvent{Button: MouseButtonLeft, Pressed: true}},
		gotEvent{btn: &MouseButtonEvent{Button: MouseButtonLeft, Pressed: false}},
	)
	time.Sleep(20 * time.Millisecond)

	if err := b.Inject(MouseScrollEvent{DY: 3}); err != nil {
		t.Fatalf("Inject scroll: %v", err)
	}
	want = append(want, gotEvent{scrollY: 3, isScroll: true})

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
	t.Logf("injected %d events, captured %d (Suppress active throughout, per 5.2 this must not block capture)", len(want), len(got))
	if len(got) < len(want) {
		t.Fatalf("captured only %d of %d events — check for \"sem mapeamento HID conhecido\" warnings above", len(got), len(want))
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
		case w.isScroll:
			if !g.isScroll || g.scrollY != w.scrollY {
				t.Fatalf("event %d: want scroll dy=%d, got %+v", i, w.scrollY, g)
			}
		}
	}
}

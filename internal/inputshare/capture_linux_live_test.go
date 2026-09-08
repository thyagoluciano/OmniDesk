//go:build linux

package inputshare

import (
	"context"
	"os"
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

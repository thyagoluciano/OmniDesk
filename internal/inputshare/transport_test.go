package inputshare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// testServer wires up a bare Accept-based receiver, recording every
// injected event, for exercising the transport without any platform
// capture/injection backend.
type testServer struct {
	mu       sync.Mutex
	received []Event
	conn     *Conn
	connCh   chan *Conn
}

func newTestServer() (*httptest.Server, *testServer) {
	ts := &testServer{connCh: make(chan *Conn, 1)}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/input/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := Accept(w, r, "client-peer")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		conn.SetReceiverHandlers(func(ev Event) error {
			ts.mu.Lock()
			ts.received = append(ts.received, ev)
			ts.mu.Unlock()
			return nil
		}, nil)
		conn.Start()
		ts.connCh <- conn
	})
	return httptest.NewServer(mux), ts
}

func (ts *testServer) events() []Event {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]Event, len(ts.received))
	copy(out, ts.received)
	return out
}

func dialTestSender(t *testing.T, srv *httptest.Server) *Conn {
	t.Helper()
	addr := strings.TrimPrefix(srv.URL, "http://")
	conn, err := DialSender(context.Background(), addr, "sender-id", "token", "server-peer")
	if err != nil {
		t.Fatalf("DialSender failed: %v", err)
	}
	conn.SetSenderHandlers(nil)
	conn.Start()
	return conn
}

func TestTransportPreservesKeyOrderUnderMoveLoad(t *testing.T) {
	srv, ts := newTestServer()
	defer srv.Close()

	sender := dialTestSender(t, srv)
	defer sender.Close()

	// Flood moves concurrently with an ordered sequence of key events —
	// specs/input-sharing-transport: "Sequência de teclas preservada sob
	// carga de movimento".
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 500; i++ {
			sender.SendMove(int16(i%5-2), int16(i%3-1))
		}
	}()

	keys := []HIDUsage{HIDKeyH, HIDKeyE, HIDKeyL, HIDKeyL, HIDKeyO}
	for _, k := range keys {
		sender.SendKey(k, true)
		sender.SendKey(k, false)
	}
	<-done

	deadline := time.Now().Add(3 * time.Second)
	for {
		var keyEvents []KeyEvent
		for _, ev := range ts.events() {
			if k, ok := ev.(KeyEvent); ok {
				keyEvents = append(keyEvents, k)
			}
		}
		if len(keyEvents) == len(keys)*2 {
			for i, k := range keys {
				if keyEvents[2*i].HID != k || !keyEvents[2*i].Pressed {
					t.Fatalf("key order mismatch at %d: %+v", i, keyEvents)
				}
				if keyEvents[2*i+1].HID != k || keyEvents[2*i+1].Pressed {
					t.Fatalf("key order mismatch at %d: %+v", i, keyEvents)
				}
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d key events, got %d", len(keys)*2, len(keyEvents))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTransportFailsafeReleasesStuckKeysOnDisconnect(t *testing.T) {
	srv, ts := newTestServer()
	defer srv.Close()

	sender := dialTestSender(t, srv)

	sender.SendKey(HIDKeyLeftControl, true)

	// Wait for the press to actually arrive before yanking the connection,
	// otherwise the test doesn't exercise what it claims to.
	deadline := time.Now().Add(2 * time.Second)
	for {
		found := false
		for _, ev := range ts.events() {
			if k, ok := ev.(KeyEvent); ok && k.HID == HIDKeyLeftControl && k.Pressed {
				found = true
			}
		}
		if found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the initial key-down to arrive")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// design.md Decision 7: killing the connection must synthesize a
	// key-up for everything still marked pressed, without relying on any
	// explicit release message getting through.
	sender.Close()

	deadline = time.Now().Add(2 * time.Second)
	for {
		var lastPressed *bool
		for _, ev := range ts.events() {
			if k, ok := ev.(KeyEvent); ok && k.HID == HIDKeyLeftControl {
				p := k.Pressed
				lastPressed = &p
			}
		}
		if lastPressed != nil && !*lastPressed {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for failsafe release; events: %+v", ts.events())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTransportCoalescesMoves(t *testing.T) {
	srv, ts := newTestServer()
	defer srv.Close()

	sender := dialTestSender(t, srv)
	defer sender.Close()

	// Queue a burst of moves back-to-back, much faster than the writer
	// goroutine can possibly drain them one at a time; only the latest
	// should ever be written (specs/input-sharing-transport: "Múltiplos
	// movimentos rápidos geram apenas o último envio"). A sentinel key
	// event — sent through the reliable, non-coalesced path — marks when
	// the burst has been fully processed, whatever order it lands in
	// relative to the (possibly still in-flight) coalesced move.
	const n = 2000
	for i := 0; i < n; i++ {
		sender.SendMove(1, 1)
	}
	sender.SendKey(HIDKeySpace, true) // sentinel: reliable events are never coalesced

	deadline := time.Now().Add(3 * time.Second)
	for {
		evs := ts.events()
		sentinelSeen := false
		moveCount := 0
		for _, ev := range evs {
			switch ev.(type) {
			case KeyEvent:
				sentinelSeen = true
			case MouseMoveEvent:
				moveCount++
			}
		}
		if sentinelSeen {
			if moveCount >= n {
				t.Fatalf("coalescing did not reduce move count: got %d of %d sent", moveCount, n)
			}
			var totalDX, totalDY int32
			for _, ev := range evs {
				if m, ok := ev.(MouseMoveEvent); ok {
					totalDX += int32(m.DX)
					totalDY += int32(m.DY)
				}
			}
			if totalDX != n || totalDY != n {
				t.Fatalf("accumulated motion did not preserve total distance: got DX=%d DY=%d, want %d", totalDX, totalDY, n)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for sentinel key event")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

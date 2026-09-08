package inputshare

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Heartbeat tuning (specs/input-sharing-transport, task 2.6). Every frame
// read refreshes the peer's read deadline; if nothing arrives (not even a
// protocol-level ping) within readTimeout, the connection is considered
// dead and torn down — which on the receiving side triggers the release-all
// failsafe (task 2.7) without depending on any message actually getting
// through.
const (
	pingInterval = 3 * time.Second
	readTimeout  = 10 * time.Second
	writeTimeout = 2 * time.Second
)

// role identifies which end of a session a Conn represents.
type role uint8

const (
	roleSender role = iota
	roleReceiver
)

// Conn is one active input-sharing WebSocket connection to a single peer.
// A node holds at most one Conn per peer at a time: either it is the
// physical source of input (roleSender, forwarding captured events) or the
// destination (roleReceiver, injecting received events).
type Conn struct {
	ws     *websocket.Conn
	peerID string
	role   role

	// Reliable, ordered queue for key/click events — never dropped
	// (specs/input-sharing-transport: "Entrega ordenada e sem perdas").
	reliableCh chan Event

	// Coalescing slot for mouse-move events — only the latest pending move
	// is kept (specs/input-sharing-transport: "Coalescing de eventos de
	// movimento").
	moveMu     sync.Mutex
	pendingMv  *MouseMoveEvent
	moveSignal chan struct{}

	closeOnce sync.Once
	closeCh   chan struct{}

	// injectedPressed tracks, on the receiving side, every HID key this
	// connection has injected as "pressed" but not yet released — the
	// failsafe releases exactly this set the moment the connection dies
	// (design.md Decision 7).
	pressMu         sync.Mutex
	injectedPressed map[HIDUsage]bool

	inject       func(Event) error // receiver role only
	onReturnReq  func()            // sender role only: peer asked for control back
	onDisconnect func()            // called exactly once, from the read loop's exit
}

func newConn(ws *websocket.Conn, peerID string, r role) *Conn {
	return &Conn{
		ws:              ws,
		peerID:          peerID,
		role:            r,
		reliableCh:      make(chan Event, 256),
		moveSignal:      make(chan struct{}, 1),
		closeCh:         make(chan struct{}),
		injectedPressed: make(map[HIDUsage]bool),
	}
}

// DialSender opens the transport-level half of an input-sharing session as
// the physical source of input: it connects out to targetAddr (host:port of
// the peer's OmniDesk HTTP server) and authenticates with this node's
// pairing credentials for that peer, exactly like every other outbound
// OmniDesk API call. The returned Conn is not yet pumping events — callers
// MUST set handlers via SetSenderHandlers and then call Start, which lets
// the Manager record the session as active before any event can possibly
// arrive (avoiding a race between connection setup and state bookkeeping).
func DialSender(ctx context.Context, targetAddr, localDeviceID, localToken, peerID string) (*Conn, error) {
	u := url.URL{Scheme: "ws", Host: targetAddr, Path: "/api/v1/input/ws"}
	header := http.Header{}
	header.Set("X-OmniDesk-Device-ID", localDeviceID)
	header.Set("X-OmniDesk-Token", localToken)

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	ws, resp, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		status := ""
		if resp != nil {
			status = fmt.Sprintf(" (HTTP %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("inputshare: failed to dial input session with %s%s: %w", targetAddr, status, err)
	}

	return newConn(ws, peerID, roleSender), nil
}

// upgrader is shared by every accepted input-sharing connection. Origin
// checking is not meaningful here: the caller has already been
// authenticated by device token in Server.authMiddleware, and authorized by
// the input-control permission, before Accept is invoked.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Accept upgrades an already-authenticated, already-authorized HTTP request
// into the transport-level half of an input-sharing session where this node
// is the destination. As with DialSender, the returned Conn does not pump
// events until SetReceiverHandlers and Start are called.
func Accept(w http.ResponseWriter, r *http.Request, peerID string) (*Conn, error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("inputshare: websocket upgrade failed: %w", err)
	}

	return newConn(ws, peerID, roleReceiver), nil
}

// SetSenderHandlers wires the callback invoked when the peer asks for
// ownership back (hot-corner / RequestReturnEvent). Only meaningful on a
// roleSender Conn.
func (c *Conn) SetSenderHandlers(onReturnReq func()) {
	c.onReturnReq = onReturnReq
}

// SetReceiverHandlers wires the injection function and disconnect callback.
// Only meaningful on a roleReceiver Conn.
func (c *Conn) SetReceiverHandlers(inject func(Event) error, onDisconnect func()) {
	c.inject = inject
	c.onDisconnect = onDisconnect
}

// Start begins pumping events. Must be called exactly once, after handlers
// are wired and after the caller has recorded this Conn as the active
// session (see package doc on the race this ordering avoids).
func (c *Conn) Start() {
	go c.readLoop()
	go c.writeLoop()
}

// readLoop drains inbound frames, refreshing the read deadline on every
// frame (including control pings) so a silent connection is detected
// locally within readTimeout regardless of what the other end intended to
// send.
func (c *Conn) readLoop() {
	c.ws.SetReadDeadline(time.Now().Add(readTimeout))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	defer c.teardown()

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		c.ws.SetReadDeadline(time.Now().Add(readTimeout))

		ev, err := Decode(data)
		if err != nil {
			log.Printf("[inputshare] %v", err)
			continue
		}

		switch ev.(type) {
		case RequestReturnEvent:
			if c.onReturnReq != nil {
				c.onReturnReq()
			}
		case ReleaseAllEvent:
			if c.role == roleReceiver {
				c.releaseAllInjected()
			}
		default:
			if c.role == roleReceiver && c.inject != nil {
				c.trackAndInject(ev)
			}
		}
	}
}

// trackAndInject maintains injectedPressed (for the disconnect failsafe)
// before handing the event to the platform backend.
func (c *Conn) trackAndInject(ev Event) {
	if key, ok := ev.(KeyEvent); ok {
		c.pressMu.Lock()
		if key.Pressed {
			c.injectedPressed[key.HID] = true
		} else {
			delete(c.injectedPressed, key.HID)
		}
		c.pressMu.Unlock()
	}
	if err := c.inject(ev); err != nil {
		log.Printf("[inputshare] injection failed: %v", err)
	}
}

// releaseAllInjected is the disconnect failsafe (design.md Decision 7): it
// synthesizes a key-up for everything this connection ever injected as
// "pressed" without a matching release, so a peer that vanishes mid-key
// never leaves a modifier stuck on the destination machine.
func (c *Conn) releaseAllInjected() {
	c.pressMu.Lock()
	held := make([]HIDUsage, 0, len(c.injectedPressed))
	for hid := range c.injectedPressed {
		held = append(held, hid)
	}
	c.injectedPressed = make(map[HIDUsage]bool)
	c.pressMu.Unlock()

	for _, hid := range held {
		if err := c.inject(KeyEvent{HID: hid, Pressed: false}); err != nil {
			log.Printf("[inputshare] failsafe release of HID 0x%02X failed: %v", hid, err)
		}
	}
	if len(held) > 0 {
		log.Printf("[inputshare] failsafe: released %d stuck key(s) after connection loss", len(held))
	}
}

// writeLoop is the single writer for this connection (required by
// gorilla/websocket) and implements the priority discipline: the reliable
// queue always drains before a coalesced move is written, so a burst of
// mouse motion never delays a key event that is already queued
// (specs/input-sharing-transport).
func (c *Conn) writeLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer c.teardown()

	for {
		// Priority pass: a plain `select` with both reliableCh and
		// moveSignal as cases would let Go's pseudo-random case choice
		// undermine the priority guarantee (specs/input-sharing-transport:
		// "Entrega ordenada e sem perdas"), so the reliable queue is
		// always drained first, non-blockingly, before anything else is
		// even considered.
		select {
		case <-c.closeCh:
			return
		case ev := <-c.reliableCh:
			if !c.writeEvent(ev) {
				return
			}
			continue
		default:
		}

		select {
		case <-c.closeCh:
			return

		case ev := <-c.reliableCh:
			if !c.writeEvent(ev) {
				return
			}
			continue

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			continue

		case <-c.moveSignal:
			// fall through to drain the pending move below
		}

		c.moveMu.Lock()
		mv := c.pendingMv
		c.pendingMv = nil
		c.moveMu.Unlock()
		if mv != nil {
			if !c.writeEvent(*mv) {
				return
			}
		}
	}
}

func (c *Conn) writeEvent(ev Event) bool {
	c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := c.ws.WriteMessage(websocket.BinaryMessage, ev.Encode()); err != nil {
		return false
	}
	return true
}

func (c *Conn) teardown() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		_ = c.ws.Close()
		if c.role == roleReceiver {
			c.releaseAllInjected()
		}
		if c.onDisconnect != nil {
			c.onDisconnect()
		}
	})
}

// Close ends the session cleanly. Callers that are ending a sender session
// on purpose (hotkey/hot-corner/toggle) should call SendReleaseAll first so
// the peer does not have to wait for the heartbeat timeout in the common,
// non-crash case.
func (c *Conn) Close() {
	c.teardown()
}

// enqueue routes an event through the coalescing or reliable path
// according to its type.
func (c *Conn) enqueue(ev Event) {
	if ev.Type().IsCoalescable() {
		mv := ev.(MouseMoveEvent)
		c.moveMu.Lock()
		c.pendingMv = &mv
		c.moveMu.Unlock()
		select {
		case c.moveSignal <- struct{}{}:
		default:
		}
		return
	}

	select {
	case c.reliableCh <- ev:
	case <-c.closeCh:
	}
}

func (c *Conn) SendMove(dx, dy int16)            { c.enqueue(MouseMoveEvent{DX: dx, DY: dy}) }
func (c *Conn) SendWarp(x, y uint16)             { c.enqueue(MouseWarpEvent{X: x, Y: y}) }
func (c *Conn) SendButton(b MouseButton, p bool) { c.enqueue(MouseButtonEvent{Button: b, Pressed: p}) }
func (c *Conn) SendScroll(dx, dy int16)          { c.enqueue(MouseScrollEvent{DX: dx, DY: dy}) }
func (c *Conn) SendReleaseAll()                  { c.enqueue(ReleaseAllEvent{}) }
func (c *Conn) SendRequestReturn()               { c.enqueue(RequestReturnEvent{}) }

func (c *Conn) SendKey(hid HIDUsage, pressed bool) {
	c.enqueue(KeyEvent{HID: hid, Pressed: pressed})
}

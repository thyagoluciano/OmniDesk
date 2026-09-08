// Package inputshare implements KVM-style mouse/keyboard sharing between
// paired OmniDesk nodes: a persistent transport, a per-pair input-ownership
// state machine, a screen-adjacency layout model, and platform-specific
// capture/injection backends.
package inputshare

import (
	"encoding/binary"
	"fmt"
)

// EventType identifies the wire format of an Event.
type EventType uint8

const (
	// EventMouseMove carries a relative pointer motion delta, used while a
	// session is actively controlling a remote node (after the initial warp).
	EventMouseMove EventType = iota + 1
	// EventMouseWarp carries an absolute pointer position, in the
	// destination's own pixel space, used once when input ownership
	// transitions to a node so the cursor reappears at the right spot. The
	// sender computes this position via Layout.Cross, which already applies
	// the proportional resolution scaling (see design.md Decision 5) — the
	// receiver just places the cursor exactly where it is told.
	EventMouseWarp
	// EventMouseButton carries a mouse button press/release.
	EventMouseButton
	// EventMouseScroll carries a scroll wheel delta.
	EventMouseScroll
	// EventKey carries a keyboard key press/release, identified by its
	// canonical HID usage code (see hid.go), never by interpreted character.
	EventKey
	// EventReleaseAll instructs the receiver to release every key it
	// currently considers pressed. Sent by the origin node as part of the
	// disconnect failsafe (design.md Decision 7).
	EventReleaseAll
	// EventPing/EventPong implement the transport heartbeat used to detect
	// a dead connection before the OS-level TCP timeout would.
	EventPing
	EventPong
	// EventRequestReturn is sent by the node currently receiving injected
	// input back to the sending node, asking it to reclaim ownership. It is
	// how the hot-corner escape mechanism works: the receiving node notices
	// its own (injected) cursor hit the reserved corner and asks the
	// physical origin to take control back (design.md Decision, "hot corner
	// fixo de retorno").
	EventRequestReturn
)

// Event is anything that can be framed onto the input-sharing WebSocket.
type Event interface {
	Type() EventType
	Encode() []byte
}

// MouseMoveEvent is a relative pointer motion.
type MouseMoveEvent struct {
	DX, DY int16
}

func (MouseMoveEvent) Type() EventType { return EventMouseMove }

func (e MouseMoveEvent) Encode() []byte {
	b := make([]byte, 5)
	b[0] = byte(EventMouseMove)
	binary.BigEndian.PutUint16(b[1:3], uint16(e.DX))
	binary.BigEndian.PutUint16(b[3:5], uint16(e.DY))
	return b
}

// MouseWarpEvent is an absolute pointer position in destination pixels.
type MouseWarpEvent struct {
	X, Y uint16
}

func (MouseWarpEvent) Type() EventType { return EventMouseWarp }

func (e MouseWarpEvent) Encode() []byte {
	b := make([]byte, 5)
	b[0] = byte(EventMouseWarp)
	binary.BigEndian.PutUint16(b[1:3], e.X)
	binary.BigEndian.PutUint16(b[3:5], e.Y)
	return b
}

// MouseButton identifies which physical button an event refers to.
type MouseButton uint8

const (
	MouseButtonLeft MouseButton = iota + 1
	MouseButtonRight
	MouseButtonMiddle
)

// MouseButtonEvent is a mouse button press or release.
type MouseButtonEvent struct {
	Button  MouseButton
	Pressed bool
}

func (MouseButtonEvent) Type() EventType { return EventMouseButton }

func (e MouseButtonEvent) Encode() []byte {
	return []byte{byte(EventMouseButton), byte(e.Button), boolByte(e.Pressed)}
}

// MouseScrollEvent is a scroll wheel delta (positive = down/right).
type MouseScrollEvent struct {
	DX, DY int16
}

func (MouseScrollEvent) Type() EventType { return EventMouseScroll }

func (e MouseScrollEvent) Encode() []byte {
	b := make([]byte, 5)
	b[0] = byte(EventMouseScroll)
	binary.BigEndian.PutUint16(b[1:3], uint16(e.DX))
	binary.BigEndian.PutUint16(b[3:5], uint16(e.DY))
	return b
}

// KeyEvent is a keyboard key press or release, identified by its canonical
// HID usage code so it survives translation between different native
// keyboard layouts on the two ends (see hid.go).
type KeyEvent struct {
	HID     HIDUsage
	Pressed bool
}

func (KeyEvent) Type() EventType { return EventKey }

func (e KeyEvent) Encode() []byte {
	b := make([]byte, 4)
	b[0] = byte(EventKey)
	binary.BigEndian.PutUint16(b[1:3], uint16(e.HID))
	b[3] = boolByte(e.Pressed)
	return b
}

// ReleaseAllEvent asks the receiver to synthesize key-up for every key it
// currently believes is held down.
type ReleaseAllEvent struct{}

func (ReleaseAllEvent) Type() EventType { return EventReleaseAll }
func (ReleaseAllEvent) Encode() []byte  { return []byte{byte(EventReleaseAll)} }

// PingEvent/PongEvent carry a nonce that the receiver must echo back
// unchanged, used to measure round-trip time and detect dead connections.
type PingEvent struct{ Nonce uint32 }

func (PingEvent) Type() EventType { return EventPing }

func (e PingEvent) Encode() []byte {
	b := make([]byte, 5)
	b[0] = byte(EventPing)
	binary.BigEndian.PutUint32(b[1:5], e.Nonce)
	return b
}

type PongEvent struct{ Nonce uint32 }

func (PongEvent) Type() EventType { return EventPong }

func (e PongEvent) Encode() []byte {
	b := make([]byte, 5)
	b[0] = byte(EventPong)
	binary.BigEndian.PutUint32(b[1:5], e.Nonce)
	return b
}

// RequestReturnEvent carries no payload beyond its type byte.
type RequestReturnEvent struct{}

func (RequestReturnEvent) Type() EventType { return EventRequestReturn }
func (RequestReturnEvent) Encode() []byte  { return []byte{byte(EventRequestReturn)} }

func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// clampDelta16 clamps an int32 motion or scroll delta to the int16 range.
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

// Decode parses a single framed Event from raw WebSocket message bytes.
func Decode(data []byte) (Event, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("inputshare: empty event frame")
	}

	switch EventType(data[0]) {
	case EventMouseMove:
		if len(data) < 5 {
			return nil, fmt.Errorf("inputshare: short MouseMove frame (%d bytes)", len(data))
		}
		return MouseMoveEvent{
			DX: int16(binary.BigEndian.Uint16(data[1:3])),
			DY: int16(binary.BigEndian.Uint16(data[3:5])),
		}, nil

	case EventMouseWarp:
		if len(data) < 5 {
			return nil, fmt.Errorf("inputshare: short MouseWarp frame (%d bytes)", len(data))
		}
		return MouseWarpEvent{
			X: binary.BigEndian.Uint16(data[1:3]),
			Y: binary.BigEndian.Uint16(data[3:5]),
		}, nil

	case EventMouseButton:
		if len(data) < 3 {
			return nil, fmt.Errorf("inputshare: short MouseButton frame (%d bytes)", len(data))
		}
		return MouseButtonEvent{
			Button:  MouseButton(data[1]),
			Pressed: data[2] != 0,
		}, nil

	case EventMouseScroll:
		if len(data) < 5 {
			return nil, fmt.Errorf("inputshare: short MouseScroll frame (%d bytes)", len(data))
		}
		return MouseScrollEvent{
			DX: int16(binary.BigEndian.Uint16(data[1:3])),
			DY: int16(binary.BigEndian.Uint16(data[3:5])),
		}, nil

	case EventKey:
		if len(data) < 4 {
			return nil, fmt.Errorf("inputshare: short Key frame (%d bytes)", len(data))
		}
		return KeyEvent{
			HID:     HIDUsage(binary.BigEndian.Uint16(data[1:3])),
			Pressed: data[3] != 0,
		}, nil

	case EventReleaseAll:
		return ReleaseAllEvent{}, nil

	case EventRequestReturn:
		return RequestReturnEvent{}, nil

	case EventPing:
		if len(data) < 5 {
			return nil, fmt.Errorf("inputshare: short Ping frame (%d bytes)", len(data))
		}
		return PingEvent{Nonce: binary.BigEndian.Uint32(data[1:5])}, nil

	case EventPong:
		if len(data) < 5 {
			return nil, fmt.Errorf("inputshare: short Pong frame (%d bytes)", len(data))
		}
		return PongEvent{Nonce: binary.BigEndian.Uint32(data[1:5])}, nil

	default:
		return nil, fmt.Errorf("inputshare: unknown event type %d", data[0])
	}
}

// IsCoalescable reports whether an event type may be superseded by a more
// recent event of the same type still waiting to be sent (see
// specs/input-sharing-transport: "Coalescing de eventos de movimento").
func (t EventType) IsCoalescable() bool {
	return t == EventMouseMove
}

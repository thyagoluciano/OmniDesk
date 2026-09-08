package inputshare

import (
	"reflect"
	"testing"
)

func TestEventEncodeDecodeRoundTrip(t *testing.T) {
	cases := []Event{
		MouseMoveEvent{DX: -120, DY: 340},
		MouseWarpEvent{X: 1919, Y: 0},
		MouseButtonEvent{Button: MouseButtonRight, Pressed: true},
		MouseButtonEvent{Button: MouseButtonLeft, Pressed: false},
		MouseScrollEvent{DX: 0, DY: -3},
		KeyEvent{HID: HIDKeyA, Pressed: true},
		KeyEvent{HID: HIDKeyLeftControl, Pressed: false},
		ReleaseAllEvent{},
		RequestReturnEvent{},
		PingEvent{Nonce: 0xDEADBEEF},
		PongEvent{Nonce: 42},
	}

	for _, want := range cases {
		encoded := want.Encode()
		got, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(%#v) failed: %v", want, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("round trip mismatch: sent %#v, got %#v", want, got)
		}
	}
}

func TestDecodeRejectsEmptyAndUnknown(t *testing.T) {
	if _, err := Decode(nil); err == nil {
		t.Error("expected error decoding empty frame")
	}
	if _, err := Decode([]byte{0xFF}); err == nil {
		t.Error("expected error decoding unknown event type")
	}
	if _, err := Decode([]byte{byte(EventMouseMove), 0x01}); err == nil {
		t.Error("expected error decoding truncated MouseMove frame")
	}
}

func TestOnlyMouseMoveIsCoalescable(t *testing.T) {
	if !EventMouseMove.IsCoalescable() {
		t.Error("EventMouseMove should be coalescable")
	}
	for _, tp := range []EventType{EventMouseWarp, EventMouseButton, EventMouseScroll, EventKey, EventReleaseAll, EventRequestReturn} {
		if tp.IsCoalescable() {
			t.Errorf("event type %v should not be coalescable", tp)
		}
	}
}

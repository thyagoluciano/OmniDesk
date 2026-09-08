//go:build darwin

package inputshare

import "testing"

// evdevHIDSetSnapshot is a literal copy of the set of HIDUsage values
// covered by keymap_linux.go's evdevToHID table. It has to be a snapshot,
// not a cross-package reference: keymap_linux.go carries a "//go:build
// linux" tag and is never compiled into a darwin build, so this darwin test
// cannot import it directly. Keeping this list in sync with
// keymap_linux.go's evdevToHID is a manual step (both tables changed
// together, on the same task, from the same design.md coverage
// requirement).
var evdevHIDSetSnapshot = []HIDUsage{
	HIDKeyEscape,
	HIDKey1, HIDKey2, HIDKey3, HIDKey4, HIDKey5, HIDKey6, HIDKey7, HIDKey8, HIDKey9, HIDKey0,
	HIDKeyMinus, HIDKeyEqual, HIDKeyBackspace, HIDKeyTab,
	HIDKeyQ, HIDKeyW, HIDKeyE, HIDKeyR, HIDKeyT, HIDKeyY, HIDKeyU, HIDKeyI, HIDKeyO, HIDKeyP,
	HIDKeyLeftBracket, HIDKeyRightBracket, HIDKeyEnter, HIDKeyLeftControl,
	HIDKeyA, HIDKeyS, HIDKeyD, HIDKeyF, HIDKeyG, HIDKeyH, HIDKeyJ, HIDKeyK, HIDKeyL,
	HIDKeySemicolon, HIDKeyQuote, HIDKeyGrave, HIDKeyLeftShift, HIDKeyBackslash,
	HIDKeyZ, HIDKeyX, HIDKeyC, HIDKeyV, HIDKeyB, HIDKeyN, HIDKeyM,
	HIDKeyComma, HIDKeyPeriod, HIDKeySlash, HIDKeyRightShift, HIDKeyKeypadStar, HIDKeyLeftAlt,
	HIDKeySpace, HIDKeyCapsLock,
	HIDKeyF1, HIDKeyF2, HIDKeyF3, HIDKeyF4, HIDKeyF5, HIDKeyF6, HIDKeyF7, HIDKeyF8, HIDKeyF9, HIDKeyF10,
	HIDKeyNumLock, HIDKeyScrollLock,
	HIDKeyKeypad7, HIDKeyKeypad8, HIDKeyKeypad9, HIDKeyKeypadMinus,
	HIDKeyKeypad4, HIDKeyKeypad5, HIDKeyKeypad6, HIDKeyKeypadPlus,
	HIDKeyKeypad1, HIDKeyKeypad2, HIDKeyKeypad3, HIDKeyKeypad0, HIDKeyKeypadPeriod,
	HIDKeyNonUSBackslash,
	HIDKeyF11, HIDKeyF12,
	HIDKeyKeypadEnter, HIDKeyRightControl, HIDKeyKeypadSlash,
	HIDKeyPrintScreen, HIDKeyRightAlt,
	HIDKeyHome, HIDKeyUp, HIDKeyPageUp,
	HIDKeyLeft, HIDKeyRight,
	HIDKeyEnd, HIDKeyDown, HIDKeyPageDown,
	HIDKeyInsert, HIDKeyDelete,
	HIDKeyPause,
	HIDKeyLeftGUI, HIDKeyRightGUI, HIDKeyApplication,
}

// darwinKnownGaps lists HID usages evdevHIDSetSnapshot covers that
// vkToHID deliberately does not (see keymap_darwin.go's doc comment on
// HIDKeyApplication). Kept as an explicit allowlist so any *other* missing
// key still fails the test loudly.
var darwinKnownGaps = map[HIDUsage]bool{
	HIDKeyApplication: true,
}

func TestVKToHIDCoversSameKeySetAsEvdev(t *testing.T) {
	have := make(map[HIDUsage]bool, len(vkToHID))
	for _, e := range vkToHID {
		have[e.HID] = true
	}

	for _, hid := range evdevHIDSetSnapshot {
		if darwinKnownGaps[hid] {
			if have[hid] {
				t.Errorf("HID 0x%02X is listed as a known gap but vkToHID actually covers it now; remove it from darwinKnownGaps", hid)
			}
			continue
		}
		if !have[hid] {
			t.Errorf("HID 0x%02X (covered by keymap_linux.go's evdevToHID) has no vkToHID counterpart", hid)
		}
	}
}

func TestBuildKeycodeToHIDMap(t *testing.T) {
	m := buildKeycodeToHIDMap()
	if len(m) != len(vkToHID) {
		t.Fatalf("buildKeycodeToHIDMap: got %d entries, want %d (duplicate VK in table?)", len(m), len(vkToHID))
	}
	if hid, ok := m[0x00]; !ok || hid != HIDKeyA {
		t.Errorf("VK 0x00: want HIDKeyA, got %v (ok=%v)", hid, ok)
	}
}

func TestBuildHIDToKeycodeMap(t *testing.T) {
	m := buildHIDToKeycodeMap()
	if vk, ok := m[HIDKeyA]; !ok || vk != 0x00 {
		t.Errorf("HIDKeyA: want VK 0x00, got %v (ok=%v)", vk, ok)
	}
	// panicHotkeyCombo (manager.go) must survive the round trip: it is the
	// one hotkey that MUST work even with no user configuration (task 7.3).
	for _, hid := range []HIDUsage{HIDKeyLeftControl, HIDKeyLeftAlt, HIDKeyLeftShift, HIDKeyEscape} {
		if _, ok := m[hid]; !ok {
			t.Errorf("panicHotkeyCombo member HID 0x%02X has no macOS keycode mapping", hid)
		}
	}
}

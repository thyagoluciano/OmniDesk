//go:build darwin

package inputshare

// vkToHID is the macOS virtual keycode (kVK_* constants, historically from
// Carbon/HIToolbox/Events.h — no supported public header to import without
// Carbon, so these are the documented literal values) to canonical HID
// Usage Code table.
//
// Unlike keymap_linux.go's evdevToHID, this is not derived by formula:
// macOS virtual keycodes are an arbitrary legacy numbering with no fixed
// offset from HID Usage Codes (design.md Decision 4), so every entry here
// is a hand-verified pairing, covering the same key set keymap_linux.go
// covers (letters, digits, standard punctuation, modifiers, function keys,
// arrows, common navigation/editing keys).
//
// One documented gap: HIDKeyApplication (the PC "Menu" key) has no
// confirmed macOS virtual keycode — Apple's own keyboards never shipped
// one, and unlike PrintScreen/ScrollLock/Pause (which Microsoft's own Mac
// Remote Desktop documentation maps to F13/F14/F15), there is no equally
// authoritative mapping to reuse. Left out rather than guessed, per the
// same rule this table follows for any untranslatable key: skip it, don't
// send a fabricated value.
var vkToHID = []struct {
	VK  byte
	HID HIDUsage
}{
	// Letters.
	{0x00, HIDKeyA}, {0x0B, HIDKeyB}, {0x08, HIDKeyC}, {0x02, HIDKeyD},
	{0x0E, HIDKeyE}, {0x03, HIDKeyF}, {0x05, HIDKeyG}, {0x04, HIDKeyH},
	{0x22, HIDKeyI}, {0x26, HIDKeyJ}, {0x28, HIDKeyK}, {0x25, HIDKeyL},
	{0x2E, HIDKeyM}, {0x2D, HIDKeyN}, {0x1F, HIDKeyO}, {0x23, HIDKeyP},
	{0x0C, HIDKeyQ}, {0x0F, HIDKeyR}, {0x01, HIDKeyS}, {0x11, HIDKeyT},
	{0x20, HIDKeyU}, {0x09, HIDKeyV}, {0x0D, HIDKeyW}, {0x07, HIDKeyX},
	{0x10, HIDKeyY}, {0x06, HIDKeyZ},

	// Top-row digits.
	{0x12, HIDKey1}, {0x13, HIDKey2}, {0x14, HIDKey3}, {0x15, HIDKey4},
	{0x17, HIDKey5}, {0x16, HIDKey6}, {0x1A, HIDKey7}, {0x1C, HIDKey8},
	{0x19, HIDKey9}, {0x1D, HIDKey0},

	// Controls and punctuation.
	{0x24, HIDKeyEnter},          // kVK_Return
	{0x35, HIDKeyEscape},         // kVK_Escape
	{0x33, HIDKeyBackspace},      // kVK_Delete (the backspace key)
	{0x30, HIDKeyTab},            // kVK_Tab
	{0x31, HIDKeySpace},          // kVK_Space
	{0x1B, HIDKeyMinus},          // kVK_ANSI_Minus
	{0x18, HIDKeyEqual},          // kVK_ANSI_Equal
	{0x21, HIDKeyLeftBracket},    // kVK_ANSI_LeftBracket
	{0x1E, HIDKeyRightBracket},   // kVK_ANSI_RightBracket
	{0x2A, HIDKeyBackslash},      // kVK_ANSI_Backslash
	{0x29, HIDKeySemicolon},      // kVK_ANSI_Semicolon
	{0x27, HIDKeyQuote},          // kVK_ANSI_Quote
	{0x32, HIDKeyGrave},          // kVK_ANSI_Grave
	{0x2B, HIDKeyComma},          // kVK_ANSI_Comma
	{0x2F, HIDKeyPeriod},         // kVK_ANSI_Period
	{0x2C, HIDKeySlash},          // kVK_ANSI_Slash
	{0x39, HIDKeyCapsLock},       // kVK_CapsLock
	{0x0A, HIDKeyNonUSBackslash}, // kVK_ISO_Section (ISO extra key)

	// Modifiers (left/right are physically distinct virtual keycodes on
	// macOS, unlike evdev's shared-then-offset scheme).
	{0x3B, HIDKeyLeftControl}, {0x3E, HIDKeyRightControl},
	{0x38, HIDKeyLeftShift}, {0x3C, HIDKeyRightShift},
	{0x3A, HIDKeyLeftAlt}, {0x3D, HIDKeyRightAlt}, // kVK_Option / kVK_RightOption
	{0x37, HIDKeyLeftGUI}, {0x36, HIDKeyRightGUI}, // kVK_Command / kVK_RightCommand

	// Function keys F1-F12.
	{0x7A, HIDKeyF1}, {0x78, HIDKeyF2}, {0x63, HIDKeyF3}, {0x76, HIDKeyF4},
	{0x60, HIDKeyF5}, {0x61, HIDKeyF6}, {0x62, HIDKeyF7}, {0x64, HIDKeyF8},
	{0x65, HIDKeyF9}, {0x6D, HIDKeyF10}, {0x67, HIDKeyF11}, {0x6F, HIDKeyF12},

	// Navigation cluster and system keys. PrintScreen/ScrollLock/Pause have
	// no dedicated key on Apple keyboards; F13/F14/F15 is the mapping
	// Microsoft's own Remote Desktop client for Mac documents for exactly
	// this Windows-key-on-a-Mac-keyboard scenario.
	{0x69, HIDKeyPrintScreen}, // kVK_F13
	{0x6B, HIDKeyScrollLock},  // kVK_F14
	{0x71, HIDKeyPause},       // kVK_F15
	{0x72, HIDKeyInsert},      // kVK_Help (occupies Insert's position on Apple keyboards)
	{0x73, HIDKeyHome},
	{0x74, HIDKeyPageUp},
	{0x75, HIDKeyDelete}, // kVK_ForwardDelete (the forward-delete key, distinct from Backspace)
	{0x77, HIDKeyEnd},
	{0x79, HIDKeyPageDown},
	{0x7B, HIDKeyLeft}, {0x7C, HIDKeyRight}, {0x7D, HIDKeyDown}, {0x7E, HIDKeyUp},

	// Numeric keypad.
	{0x52, HIDKeyKeypad0}, {0x53, HIDKeyKeypad1}, {0x54, HIDKeyKeypad2},
	{0x55, HIDKeyKeypad3}, {0x56, HIDKeyKeypad4}, {0x57, HIDKeyKeypad5},
	{0x58, HIDKeyKeypad6}, {0x59, HIDKeyKeypad7}, {0x5B, HIDKeyKeypad8},
	{0x5C, HIDKeyKeypad9},
	{0x41, HIDKeyKeypadPeriod}, // kVK_ANSI_KeypadDecimal
	{0x43, HIDKeyKeypadStar},   // kVK_ANSI_KeypadMultiply
	{0x4B, HIDKeyKeypadSlash},  // kVK_ANSI_KeypadDivide
	{0x45, HIDKeyKeypadPlus},   // kVK_ANSI_KeypadPlus
	{0x4E, HIDKeyKeypadMinus},  // kVK_ANSI_KeypadMinus
	{0x4C, HIDKeyKeypadEnter},  // kVK_ANSI_KeypadEnter
	{0x47, HIDKeyNumLock},      // kVK_ANSI_KeypadClear (occupies NumLock's position on Apple keypads)
}

func buildKeycodeToHIDMap() map[byte]HIDUsage {
	m := make(map[byte]HIDUsage, len(vkToHID))
	for _, e := range vkToHID {
		m[e.VK] = e.HID
	}
	return m
}

func buildHIDToKeycodeMap() map[HIDUsage]byte {
	m := make(map[HIDUsage]byte, len(vkToHID))
	for _, e := range vkToHID {
		m[e.HID] = e.VK
	}
	return m
}

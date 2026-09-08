package inputshare

// HIDUsage is a canonical, OS-independent key identity, taken from the USB
// HID Usage Tables specification, Usage Page 0x07 (Keyboard/Keypad Page).
//
// Every capture backend (X11, Windows, macOS, Wayland) MUST translate its
// native key code to one of these values before framing a KeyEvent, and
// every injection backend MUST translate it back to whatever native code
// produces that physical key on the local keyboard layout. This is what
// lets a key press survive a trip between, say, a Brazilian ABNT2 keyboard
// and a US keyboard without the wrong character being typed on the other
// side (see design.md Decision 3): we transmit "the physical key labeled
// A", never "the character 'a'" or a layout-specific scancode.
type HIDUsage uint16

// Letters.
const (
	HIDKeyA HIDUsage = 0x04 + iota
	HIDKeyB
	HIDKeyC
	HIDKeyD
	HIDKeyE
	HIDKeyF
	HIDKeyG
	HIDKeyH
	HIDKeyI
	HIDKeyJ
	HIDKeyK
	HIDKeyL
	HIDKeyM
	HIDKeyN
	HIDKeyO
	HIDKeyP
	HIDKeyQ
	HIDKeyR
	HIDKeyS
	HIDKeyT
	HIDKeyU
	HIDKeyV
	HIDKeyW
	HIDKeyX
	HIDKeyY
	HIDKeyZ
)

// Top-row digits (not numpad).
const (
	HIDKey1 HIDUsage = 0x1E + iota
	HIDKey2
	HIDKey3
	HIDKey4
	HIDKey5
	HIDKey6
	HIDKey7
	HIDKey8
	HIDKey9
	HIDKey0
)

// Controls and punctuation.
const (
	HIDKeyEnter        HIDUsage = 0x28
	HIDKeyEscape       HIDUsage = 0x29
	HIDKeyBackspace    HIDUsage = 0x2A
	HIDKeyTab          HIDUsage = 0x2B
	HIDKeySpace        HIDUsage = 0x2C
	HIDKeyMinus        HIDUsage = 0x2D // - _
	HIDKeyEqual        HIDUsage = 0x2E // = +
	HIDKeyLeftBracket  HIDUsage = 0x2F // [ {
	HIDKeyRightBracket HIDUsage = 0x30 // ] }
	HIDKeyBackslash    HIDUsage = 0x31 // \ |
	HIDKeySemicolon    HIDUsage = 0x33 // ; :
	HIDKeyQuote        HIDUsage = 0x34 // ' "
	HIDKeyGrave        HIDUsage = 0x35 // ` ~
	HIDKeyComma        HIDUsage = 0x36 // , <
	HIDKeyPeriod       HIDUsage = 0x37 // . >
	HIDKeySlash        HIDUsage = 0x38 // / ?
	HIDKeyCapsLock     HIDUsage = 0x39
)

// Function keys F1-F12 (0x3A-0x45) and F13-F24 (0x68-0x73).
const (
	HIDKeyF1 HIDUsage = 0x3A + iota
	HIDKeyF2
	HIDKeyF3
	HIDKeyF4
	HIDKeyF5
	HIDKeyF6
	HIDKeyF7
	HIDKeyF8
	HIDKeyF9
	HIDKeyF10
	HIDKeyF11
	HIDKeyF12
)

// Navigation cluster and system keys.
const (
	HIDKeyPrintScreen HIDUsage = 0x46
	HIDKeyScrollLock  HIDUsage = 0x47
	HIDKeyPause       HIDUsage = 0x48
	HIDKeyInsert      HIDUsage = 0x49
	HIDKeyHome        HIDUsage = 0x4A
	HIDKeyPageUp      HIDUsage = 0x4B
	HIDKeyDelete      HIDUsage = 0x4C
	HIDKeyEnd         HIDUsage = 0x4D
	HIDKeyPageDown    HIDUsage = 0x4E
	HIDKeyRight       HIDUsage = 0x4F
	HIDKeyLeft        HIDUsage = 0x50
	HIDKeyDown        HIDUsage = 0x51
	HIDKeyUp          HIDUsage = 0x52
)

// Numeric keypad.
const (
	HIDKeyNumLock      HIDUsage = 0x53
	HIDKeyKeypadSlash  HIDUsage = 0x54
	HIDKeyKeypadStar   HIDUsage = 0x55
	HIDKeyKeypadMinus  HIDUsage = 0x56
	HIDKeyKeypadPlus   HIDUsage = 0x57
	HIDKeyKeypadEnter  HIDUsage = 0x58
	HIDKeyKeypad1      HIDUsage = 0x59
	HIDKeyKeypad2      HIDUsage = 0x5A
	HIDKeyKeypad3      HIDUsage = 0x5B
	HIDKeyKeypad4      HIDUsage = 0x5C
	HIDKeyKeypad5      HIDUsage = 0x5D
	HIDKeyKeypad6      HIDUsage = 0x5E
	HIDKeyKeypad7      HIDUsage = 0x5F
	HIDKeyKeypad8      HIDUsage = 0x60
	HIDKeyKeypad9      HIDUsage = 0x61
	HIDKeyKeypad0      HIDUsage = 0x62
	HIDKeyKeypadPeriod HIDUsage = 0x63
)

// International/ISO extra keys. HIDKeyInternational1 is, notably, the extra
// key present on ISO keyboards (including Brazilian ABNT2) between the
// right Shift and Z, or near the right Shift, carrying "/" and "?" on
// ABNT2 — distinct from the ANSI HIDKeySlash.
const (
	HIDKeyNonUSBackslash HIDUsage = 0x64
	HIDKeyApplication    HIDUsage = 0x65 // "menu" key
	HIDKeyInternational1 HIDUsage = 0x87
)

// Modifier keys.
const (
	HIDKeyLeftControl  HIDUsage = 0xE0
	HIDKeyLeftShift    HIDUsage = 0xE1
	HIDKeyLeftAlt      HIDUsage = 0xE2
	HIDKeyLeftGUI      HIDUsage = 0xE3 // Cmd / Super / Win
	HIDKeyRightControl HIDUsage = 0xE4
	HIDKeyRightShift   HIDUsage = 0xE5
	HIDKeyRightAlt     HIDUsage = 0xE6
	HIDKeyRightGUI     HIDUsage = 0xE7
)

// IsModifier reports whether a HID usage identifies a modifier key
// (Ctrl/Shift/Alt/GUI, either side). The transport failsafe (design.md
// Decision 7) pays special attention to these: a stuck modifier is what
// makes a remote machine effectively unusable.
func (h HIDUsage) IsModifier() bool {
	return h >= HIDKeyLeftControl && h <= HIDKeyRightGUI
}

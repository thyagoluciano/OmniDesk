//go:build linux

package inputshare

// evdevToHID is the canonical Linux evdev (linux/input-event-codes.h)
// KEY_* code to USB HID usage code table — the same table the Linux
// kernel's own HID drivers use internally (drivers/hid/hid-input.c,
// hid_keyboard[]), just inverted.
//
// X11 keycodes are derived from it, not looked up per layout: virtually
// every modern Linux desktop (anything using libinput/xf86-input-evdev via
// the standard "evdev" XKB rules, which is the default everywhere) numbers
// its X11 keycodes as evdevCode+8. That fixed offset is what lets a single
// table serve every keyboard layout: the *physical* key identity (HID
// usage) never depends on which characters the active layout paints on it
// (see hid.go's package doc and design.md Decision 3).
//
// This table only lists keys present on a standard 104/105-key layout. A
// captured keycode outside it is skipped with a warning (specs
// input-capture-inject-x11: "Tradução de código de tecla... SHALL
// descartar... tecla capturada cujo keycode não tenha mapeamento
// conhecido").
var evdevToHID = []struct {
	Evdev uint16
	HID   HIDUsage
}{
	{1, HIDKeyEscape},
	{2, HIDKey1}, {3, HIDKey2}, {4, HIDKey3}, {5, HIDKey4}, {6, HIDKey5},
	{7, HIDKey6}, {8, HIDKey7}, {9, HIDKey8}, {10, HIDKey9}, {11, HIDKey0},
	{12, HIDKeyMinus}, {13, HIDKeyEqual}, {14, HIDKeyBackspace}, {15, HIDKeyTab},
	{16, HIDKeyQ}, {17, HIDKeyW}, {18, HIDKeyE}, {19, HIDKeyR}, {20, HIDKeyT},
	{21, HIDKeyY}, {22, HIDKeyU}, {23, HIDKeyI}, {24, HIDKeyO}, {25, HIDKeyP},
	{26, HIDKeyLeftBracket}, {27, HIDKeyRightBracket}, {28, HIDKeyEnter},
	{29, HIDKeyLeftControl},
	{30, HIDKeyA}, {31, HIDKeyS}, {32, HIDKeyD}, {33, HIDKeyF}, {34, HIDKeyG},
	{35, HIDKeyH}, {36, HIDKeyJ}, {37, HIDKeyK}, {38, HIDKeyL},
	{39, HIDKeySemicolon}, {40, HIDKeyQuote}, {41, HIDKeyGrave},
	{42, HIDKeyLeftShift}, {43, HIDKeyBackslash},
	{44, HIDKeyZ}, {45, HIDKeyX}, {46, HIDKeyC}, {47, HIDKeyV}, {48, HIDKeyB},
	{49, HIDKeyN}, {50, HIDKeyM}, {51, HIDKeyComma}, {52, HIDKeyPeriod}, {53, HIDKeySlash},
	{54, HIDKeyRightShift}, {55, HIDKeyKeypadStar}, {56, HIDKeyLeftAlt},
	{57, HIDKeySpace}, {58, HIDKeyCapsLock},
	{59, HIDKeyF1}, {60, HIDKeyF2}, {61, HIDKeyF3}, {62, HIDKeyF4}, {63, HIDKeyF5},
	{64, HIDKeyF6}, {65, HIDKeyF7}, {66, HIDKeyF8}, {67, HIDKeyF9}, {68, HIDKeyF10},
	{69, HIDKeyNumLock}, {70, HIDKeyScrollLock},
	{71, HIDKeyKeypad7}, {72, HIDKeyKeypad8}, {73, HIDKeyKeypad9}, {74, HIDKeyKeypadMinus},
	{75, HIDKeyKeypad4}, {76, HIDKeyKeypad5}, {77, HIDKeyKeypad6}, {78, HIDKeyKeypadPlus},
	{79, HIDKeyKeypad1}, {80, HIDKeyKeypad2}, {81, HIDKeyKeypad3}, {82, HIDKeyKeypad0},
	{83, HIDKeyKeypadPeriod},
	{86, HIDKeyNonUSBackslash}, // ISO extra key next to left Shift (KEY_102ND)
	{87, HIDKeyF11}, {88, HIDKeyF12},
	{96, HIDKeyKeypadEnter}, {97, HIDKeyRightControl}, {98, HIDKeyKeypadSlash},
	{99, HIDKeyPrintScreen}, {100, HIDKeyRightAlt},
	{102, HIDKeyHome}, {103, HIDKeyUp}, {104, HIDKeyPageUp},
	{105, HIDKeyLeft}, {106, HIDKeyRight},
	{107, HIDKeyEnd}, {108, HIDKeyDown}, {109, HIDKeyPageDown},
	{110, HIDKeyInsert}, {111, HIDKeyDelete},
	{119, HIDKeyPause},
	{125, HIDKeyLeftGUI}, {126, HIDKeyRightGUI}, {127, HIDKeyApplication},
}

// x11KeycodeOffset is added to an evdev code to get the X11 keycode under
// the "evdev" XKB rules that virtually every modern Linux desktop uses.
const x11KeycodeOffset = 8

func buildKeycodeToHIDMap() map[byte]HIDUsage {
	m := make(map[byte]HIDUsage, len(evdevToHID))
	for _, e := range evdevToHID {
		m[byte(e.Evdev+x11KeycodeOffset)] = e.HID
	}
	return m
}

func buildHIDToKeycodeMap() map[HIDUsage]byte {
	m := make(map[HIDUsage]byte, len(evdevToHID))
	for _, e := range evdevToHID {
		m[e.HID] = byte(e.Evdev + x11KeycodeOffset)
	}
	return m
}

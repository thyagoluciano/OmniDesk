//go:build darwin

package inputshare

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// purego, not cgo (design.md Decision 1): every CoreFoundation/
// ApplicationServices symbol used by the macOS backend is resolved at
// runtime via dlopen/dlsym instead of linked at build time. That is what
// lets `GOOS=darwin GOARCH=arm64 go build` produce a working binary from
// any machine, with no macOS SDK or C compiler involved.

// CGPoint mirrors CoreGraphics' CGPoint: two IEEE-754 doubles, x then y.
// purego marshals a two-float64 struct as the platform ABI expects (a
// homogeneous floating-point aggregate on both amd64 and arm64), so it can
// be passed by value to CGEventCreateMouseEvent/CGEventGetLocation exactly
// like a C caller would, without cgo.
type CGPoint struct {
	X, Y float64
}

// CGEventType values used by this package (CoreGraphics CGEventTypes.h).
// Only the subset this backend cares about is listed.
const (
	cgEventNull              = 0
	cgEventLeftMouseDown     = 1
	cgEventLeftMouseUp       = 2
	cgEventRightMouseDown    = 3
	cgEventRightMouseUp      = 4
	cgEventMouseMoved        = 5
	cgEventLeftMouseDragged  = 6
	cgEventRightMouseDragged = 7
	cgEventKeyDown           = 10
	cgEventKeyUp             = 11
	cgEventFlagsChanged      = 12
	cgEventScrollWheel       = 22
	cgEventOtherMouseDown    = 25
	cgEventOtherMouseUp      = 26
	cgEventOtherMouseDragged = 27

	// Sentinel "event types" a tap callback receives instead of a real
	// event when the OS disables the tap (design.md Risk 2).
	cgEventTapDisabledByTimeout   = 0xFFFFFFFE
	cgEventTapDisabledByUserInput = 0xFFFFFFFF
)

// CGEventField values (CoreGraphics CGEventTypes.h) this backend reads.
const (
	cgMouseEventDeltaX           = 4
	cgMouseEventDeltaY           = 5
	cgKeyboardEventKeycode       = 9
	cgScrollWheelEventDeltaAxis1 = 11 // vertical
	cgScrollWheelEventDeltaAxis2 = 12 // horizontal
)

// CGEventTapLocation/Placement/Options and the "capture everything" mask.
const (
	cgSessionEventTap       = 1
	cgHeadInsertEventTap    = 0
	cgEventTapOptionDefault = 0
)

// darwinEventMask covers every event type dispatchEvent understands
// (CGEventMaskBit(type) == 1<<type, per CoreGraphics' own macro).
var darwinEventMask = func() uint64 {
	var mask uint64
	for _, t := range []uint32{
		cgEventLeftMouseDown, cgEventLeftMouseUp,
		cgEventRightMouseDown, cgEventRightMouseUp,
		cgEventMouseMoved, cgEventLeftMouseDragged, cgEventRightMouseDragged, cgEventOtherMouseDragged,
		cgEventKeyDown, cgEventKeyUp, cgEventFlagsChanged,
		cgEventScrollWheel,
		cgEventOtherMouseDown, cgEventOtherMouseUp,
	} {
		mask |= 1 << t
	}
	return mask
}()

// darwinAPI holds every symbol the backend needs, resolved once (design.md
// Decision 2). A zero value is never used: loadDarwinAPI always returns
// either a fully-populated *darwinAPI or an error.
type darwinAPI struct {
	// CoreFoundation.
	CFRelease                     func(cf uintptr)
	CFMachPortCreateRunLoopSource func(allocator uintptr, port uintptr, order int64) uintptr
	CFRunLoopGetCurrent           func() uintptr
	CFRunLoopAddSource            func(rl uintptr, source uintptr, mode uintptr)
	CFRunLoopRun                  func()
	CFRunLoopStop                 func(rl uintptr)
	CFDictionaryCreate            func(allocator uintptr, keys, values *uintptr, numValues int64, keyCB, valCB uintptr) uintptr

	// ApplicationServices / CoreGraphics (Quartz Event Services).
	CGEventCreate                 func(source uintptr) uintptr
	CGEventTapCreate              func(tap uint32, place uint32, options uint32, eventsOfInterest uint64, callback uintptr, userInfo uintptr) uintptr
	CGEventTapEnable              func(tap uintptr, enable bool)
	CGEventPost                   func(tapLoc uint32, event uintptr)
	CGEventCreateKeyboardEvent    func(source uintptr, virtualKey uint16, keyDown bool) uintptr
	CGEventCreateMouseEvent       func(source uintptr, mouseType uint32, point CGPoint, mouseButton uint32) uintptr
	CGEventSetType                func(event uintptr, eventType uint32)
	CGEventGetIntegerValueField   func(event uintptr, field uint32) int64
	CGEventSetIntegerValueField   func(event uintptr, field uint32, value int64)
	CGEventGetLocation            func(event uintptr) CGPoint
	CGEventSetLocation            func(event uintptr, location CGPoint)
	CGMainDisplayID               func() uint32
	CGDisplayPixelsWide           func(display uint32) int64
	CGDisplayPixelsHigh           func(display uint32) int64
	AXIsProcessTrustedWithOptions func(options uintptr) bool

	// Well-known CF/AX constants. These are C global *variables* (not
	// functions), so they are resolved via Dlsym + a manual pointer read
	// (readGlobalPtr) rather than RegisterLibFunc.
	kAXTrustedCheckOptionPrompt     uintptr
	kCFRunLoopCommonModes           uintptr
	kCFBooleanTrue                  uintptr
	kCFTypeDictionaryKeyCallBacks   uintptr
	kCFTypeDictionaryValueCallBacks uintptr
}

var (
	darwinOnce     sync.Once
	darwinAPICache *darwinAPI
	darwinAPIErr   error
)

// loadDarwinAPI dlopen's CoreFoundation and ApplicationServices and
// resolves every symbol darwinAPI needs, exactly once per process. Callers
// (newDarwinBackend, and the opt-in live test) all share this cache.
func loadDarwinAPI() (*darwinAPI, error) {
	darwinOnce.Do(func() {
		darwinAPICache, darwinAPIErr = dlopenDarwinAPI()
	})
	return darwinAPICache, darwinAPIErr
}

// registerFunc is RegisterLibFunc without its panic-on-missing-symbol
// behavior (design.md's accepted trade-off: a typo'd symbol name must fail
// at Start() time with a normal error, not crash the process).
func registerFunc(fptr any, lib uintptr, name string) error {
	sym, err := purego.Dlsym(lib, name)
	if err != nil {
		return fmt.Errorf("symbol %q: %w", name, err)
	}
	purego.RegisterFunc(fptr, sym)
	return nil
}

// readGlobalPtr reads the pointer-sized value stored at a C global
// variable's own address — what a C caller gets by writing the bare
// variable name (as opposed to "&variable", which is just addr itself).
// Every CF/AX constant this package needs (kAXTrustedCheckOptionPrompt,
// kCFBooleanTrue, kCFRunLoopCommonModes) is declared in C as a pointer
// variable (e.g. "extern const CFStringRef kCFRunLoopCommonModes"), so this
// dereference is required to get the actual CFStringRef/CFBooleanRef value
// rather than the address of the variable holding it.
func readGlobalPtr(addr uintptr) uintptr {
	return *(*uintptr)(unsafe.Add(unsafe.Pointer(nil), addr))
}

func dlopenDarwinAPI() (*darwinAPI, error) {
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("inputshare/darwin: dlopen CoreFoundation: %w", err)
	}
	as, err := purego.Dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("inputshare/darwin: dlopen ApplicationServices: %w", err)
	}

	api := &darwinAPI{}
	var regErr error
	reg := func(fptr any, lib uintptr, name string) {
		if regErr != nil {
			return
		}
		regErr = registerFunc(fptr, lib, name)
	}

	reg(&api.CFRelease, cf, "CFRelease")
	reg(&api.CFMachPortCreateRunLoopSource, cf, "CFMachPortCreateRunLoopSource")
	reg(&api.CFRunLoopGetCurrent, cf, "CFRunLoopGetCurrent")
	reg(&api.CFRunLoopAddSource, cf, "CFRunLoopAddSource")
	reg(&api.CFRunLoopRun, cf, "CFRunLoopRun")
	reg(&api.CFRunLoopStop, cf, "CFRunLoopStop")
	reg(&api.CFDictionaryCreate, cf, "CFDictionaryCreate")

	reg(&api.CGEventCreate, as, "CGEventCreate")
	reg(&api.CGEventTapCreate, as, "CGEventTapCreate")
	reg(&api.CGEventTapEnable, as, "CGEventTapEnable")
	reg(&api.CGEventPost, as, "CGEventPost")
	reg(&api.CGEventCreateKeyboardEvent, as, "CGEventCreateKeyboardEvent")
	reg(&api.CGEventCreateMouseEvent, as, "CGEventCreateMouseEvent")
	reg(&api.CGEventSetType, as, "CGEventSetType")
	reg(&api.CGEventGetIntegerValueField, as, "CGEventGetIntegerValueField")
	reg(&api.CGEventSetIntegerValueField, as, "CGEventSetIntegerValueField")
	reg(&api.CGEventGetLocation, as, "CGEventGetLocation")
	reg(&api.CGEventSetLocation, as, "CGEventSetLocation")
	reg(&api.CGMainDisplayID, as, "CGMainDisplayID")
	reg(&api.CGDisplayPixelsWide, as, "CGDisplayPixelsWide")
	reg(&api.CGDisplayPixelsHigh, as, "CGDisplayPixelsHigh")
	reg(&api.AXIsProcessTrustedWithOptions, as, "AXIsProcessTrustedWithOptions")
	if regErr != nil {
		return nil, fmt.Errorf("inputshare/darwin: resolving symbol: %w", regErr)
	}

	globals := []struct {
		lib   uintptr
		name  string
		dst   *uintptr
		deref bool
	}{
		{as, "kAXTrustedCheckOptionPrompt", &api.kAXTrustedCheckOptionPrompt, true},
		{cf, "kCFRunLoopCommonModes", &api.kCFRunLoopCommonModes, true},
		{cf, "kCFBooleanTrue", &api.kCFBooleanTrue, true},
		// kCFTypeDictionaryKeyCallBacks/ValueCallBacks are themselves the
		// struct constants (not pointer variables) — C code uses them via
		// "&kCFTypeDictionaryKeyCallBacks", i.e. the symbol's own address,
		// so no dereference here.
		{cf, "kCFTypeDictionaryKeyCallBacks", &api.kCFTypeDictionaryKeyCallBacks, false},
		{cf, "kCFTypeDictionaryValueCallBacks", &api.kCFTypeDictionaryValueCallBacks, false},
	}
	for _, g := range globals {
		addr, err := purego.Dlsym(g.lib, g.name)
		if err != nil {
			return nil, fmt.Errorf("inputshare/darwin: resolving symbol %q: %w", g.name, err)
		}
		if g.deref {
			*g.dst = readGlobalPtr(addr)
		} else {
			*g.dst = addr
		}
	}

	return api, nil
}

// axOptionsWithPrompt builds the {kAXTrustedCheckOptionPrompt: true}
// CFDictionary that AXIsProcessTrustedWithOptions uses to trigger the
// system's own "OmniDesk wants to control this computer" consent dialog
// when Accessibility hasn't been granted yet (design.md Decision 5). The
// caller must CFRelease the result.
func (api *darwinAPI) axOptionsWithPrompt() uintptr {
	keys := []uintptr{api.kAXTrustedCheckOptionPrompt}
	vals := []uintptr{api.kCFBooleanTrue}
	return api.CFDictionaryCreate(0, &keys[0], &vals[0], 1, api.kCFTypeDictionaryKeyCallBacks, api.kCFTypeDictionaryValueCallBacks)
}

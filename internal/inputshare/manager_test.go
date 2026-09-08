package inputshare

import "testing"

func TestAllHeld(t *testing.T) {
	combo := []HIDUsage{HIDKeyLeftControl, HIDKeyLeftAlt, HIDKeyHome}

	held := map[HIDUsage]bool{HIDKeyLeftControl: true, HIDKeyLeftAlt: true}
	if allHeld(held, combo) {
		t.Error("expected allHeld=false while one key of the combo is still up")
	}

	held[HIDKeyHome] = true
	if !allHeld(held, combo) {
		t.Error("expected allHeld=true once every key in the combo is pressed")
	}
}

func TestHitsCorner(t *testing.T) {
	rect := ScreenRect{WidthPx: 1920, HeightPx: 1080}

	cases := []struct {
		name   string
		x, y   int
		corner Corner
		want   bool
	}{
		{"top-left hit", 0, 0, CornerTopLeft, true},
		{"top-left within tolerance", 3, 2, CornerTopLeft, true},
		{"top-left out of tolerance", 10, 0, CornerTopLeft, false},
		{"top-right hit", 1919, 0, CornerTopRight, true},
		{"bottom-left hit", 0, 1079, CornerBottomLeft, true},
		{"bottom-right hit", 1919, 1079, CornerBottomRight, true},
		{"wrong corner", 0, 0, CornerBottomRight, false},
		{"none disabled", 0, 0, CornerNone, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hitsCorner(c.x, c.y, rect, c.corner); got != c.want {
				t.Errorf("hitsCorner(%d,%d,%v) = %v, want %v", c.x, c.y, c.corner, got, c.want)
			}
		})
	}
}

func TestEdgeAt(t *testing.T) {
	m := &Manager{localRect: ScreenRect{WidthPx: 1920, HeightPx: 1080}}

	if _, _, ok := m.edgeAt(960, 540); ok {
		t.Error("expected no edge in the middle of the screen")
	}
	if edge, along, ok := m.edgeAt(0, 300); !ok || edge != EdgeLeft || along != 300 {
		t.Errorf("expected EdgeLeft at along=300, got edge=%v along=%d ok=%v", edge, along, ok)
	}
	if edge, _, ok := m.edgeAt(1919, 300); !ok || edge != EdgeRight {
		t.Errorf("expected EdgeRight, got edge=%v ok=%v", edge, ok)
	}
	if edge, _, ok := m.edgeAt(300, 0); !ok || edge != EdgeTop {
		t.Errorf("expected EdgeTop, got edge=%v ok=%v", edge, ok)
	}
	if edge, _, ok := m.edgeAt(300, 1079); !ok || edge != EdgeBottom {
		t.Errorf("expected EdgeBottom, got edge=%v ok=%v", edge, ok)
	}
}

func TestPausePeerBlocksAndClearsActiveSession(t *testing.T) {
	m := &Manager{pausedPeers: make(map[string]bool)}

	if m.isPaused("peer-1") {
		t.Fatal("expected not paused initially")
	}
	m.PausePeer("peer-1")
	if !m.isPaused("peer-1") {
		t.Error("expected peer-1 to be paused")
	}
	if m.isPaused("peer-2") {
		t.Error("pausing peer-1 must not affect peer-2 (specs: toggle é por dispositivo)")
	}

	m.ResumePeer("peer-1")
	if m.isPaused("peer-1") {
		t.Error("expected peer-1 to no longer be paused after ResumePeer")
	}
}

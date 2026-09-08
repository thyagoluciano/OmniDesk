package inputshare

import "testing"

func TestCrossProportionalScaling(t *testing.T) {
	l := NewLayout()
	l.SetNode("desktop", ScreenRect{WidthPx: 2560, HeightPx: 1440})
	l.SetNode("laptop", ScreenRect{WidthPx: 1920, HeightPx: 1080})

	if err := l.SetLink(Link{FromNode: "desktop", FromEdge: EdgeRight, ToNode: "laptop", ToEdge: EdgeLeft, Offset: 0}); err != nil {
		t.Fatalf("SetLink failed: %v", err)
	}

	// 50% down the 1440px-tall source edge should land at 50% of the
	// 1080px-tall destination edge (specs/screen-layout: "Travessia entre
	// telas de alturas diferentes").
	c, ok := l.Cross("desktop", EdgeRight, 720)
	if !ok {
		t.Fatal("expected a crossing to be found")
	}
	if c.ToNode != "laptop" || c.ToEdge != EdgeLeft {
		t.Fatalf("unexpected crossing target: %+v", c)
	}
	if c.AlongPx != 540 {
		t.Errorf("expected proportional AlongPx=540, got %d", c.AlongPx)
	}
}

func TestCrossWithOffset(t *testing.T) {
	l := NewLayout()
	l.SetNode("wide", ScreenRect{WidthPx: 2000, HeightPx: 1000})
	l.SetNode("narrow", ScreenRect{WidthPx: 1000, HeightPx: 1000})

	if err := l.SetLink(Link{FromNode: "wide", FromEdge: EdgeBottom, ToNode: "narrow", ToEdge: EdgeTop, Offset: 0}); err != nil {
		t.Fatalf("SetLink failed: %v", err)
	}

	// Offset 0 aligns the starting (top/left) corners, so position 0 on the
	// source edge maps to position 0 on the destination edge.
	c, ok := l.Cross("wide", EdgeBottom, 0)
	if !ok {
		t.Fatal("expected a crossing to be found")
	}
	if c.AlongPx != 0 {
		t.Errorf("expected AlongPx=0 at the aligned start, got %d", c.AlongPx)
	}
}

func TestCrossWithoutConfiguredNeighbor(t *testing.T) {
	l := NewLayout()
	l.SetNode("solo", ScreenRect{WidthPx: 1920, HeightPx: 1080})

	if _, ok := l.Cross("solo", EdgeLeft, 500); ok {
		t.Error("expected no crossing when no link is configured for that edge")
	}
}

func TestCrossClampsOutOfRangePosition(t *testing.T) {
	l := NewLayout()
	l.SetNode("a", ScreenRect{WidthPx: 1000, HeightPx: 1000})
	l.SetNode("b", ScreenRect{WidthPx: 500, HeightPx: 500})
	if err := l.SetLink(Link{FromNode: "a", FromEdge: EdgeTop, ToNode: "b", ToEdge: EdgeBottom, Offset: 0}); err != nil {
		t.Fatalf("SetLink failed: %v", err)
	}

	c, ok := l.Cross("a", EdgeTop, 999) // 99.9% along a 1000px edge
	if !ok {
		t.Fatal("expected a crossing to be found")
	}
	if c.AlongPx < 0 || c.AlongPx >= 500 {
		t.Errorf("expected AlongPx clamped within [0,500), got %d", c.AlongPx)
	}
}

func TestSetLinkValidation(t *testing.T) {
	l := NewLayout()
	l.SetNode("a", ScreenRect{WidthPx: 100, HeightPx: 100})

	if err := l.SetLink(Link{FromNode: "a", FromEdge: EdgeTop, ToNode: "missing", ToEdge: EdgeBottom}); err == nil {
		t.Error("expected error linking to an unknown node")
	}
	if err := l.SetLink(Link{FromNode: "a", FromEdge: EdgeTop, ToNode: "a", ToEdge: EdgeBottom}); err == nil {
		t.Error("expected error linking a node to itself")
	}
	if err := l.SetLink(Link{FromNode: "a", FromEdge: "diagonal", ToNode: "a"}); err == nil {
		t.Error("expected error for an invalid edge")
	}
}

func TestRemoveNodeDropsItsLinks(t *testing.T) {
	l := NewLayout()
	l.SetNode("a", ScreenRect{WidthPx: 100, HeightPx: 100})
	l.SetNode("b", ScreenRect{WidthPx: 100, HeightPx: 100})
	if err := l.SetLink(Link{FromNode: "a", FromEdge: EdgeRight, ToNode: "b", ToEdge: EdgeLeft}); err != nil {
		t.Fatalf("SetLink failed: %v", err)
	}

	l.RemoveNode("b")

	if _, ok := l.Cross("a", EdgeRight, 50); ok {
		t.Error("expected the link to be gone after removing its target node")
	}
}

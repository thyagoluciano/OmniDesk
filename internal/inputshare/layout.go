package inputshare

import (
	"fmt"
	"sync"
)

// Edge identifies one side of a node's logical screen rectangle.
type Edge string

const (
	EdgeTop    Edge = "top"
	EdgeRight  Edge = "right"
	EdgeBottom Edge = "bottom"
	EdgeLeft   Edge = "left"
)

// Opposite returns the edge a cursor enters through when it exits through e
// on a directly-facing neighbor (used for sensible defaults, not enforced).
func (e Edge) Opposite() Edge {
	switch e {
	case EdgeTop:
		return EdgeBottom
	case EdgeBottom:
		return EdgeTop
	case EdgeLeft:
		return EdgeRight
	case EdgeRight:
		return EdgeLeft
	default:
		return e
	}
}

func (e Edge) valid() bool {
	switch e {
	case EdgeTop, EdgeRight, EdgeBottom, EdgeLeft:
		return true
	default:
		return false
	}
}

// ScreenRect is a node's logical desktop bounding box (design.md Decision 5:
// one rectangle per PC, even when the PC has multiple physical monitors).
type ScreenRect struct {
	WidthPx  int `json:"width_px"`
	HeightPx int `json:"height_px"`
}

// Link is one directed border adjacency: crossing FromEdge on FromNode
// transitions input ownership to ToNode, reappearing on ToEdge.
type Link struct {
	FromNode string `json:"from_node"`
	FromEdge Edge   `json:"from_edge"`
	ToNode   string `json:"to_node"`
	ToEdge   Edge   `json:"to_edge"`
	// Offset aligns the two edges when their lengths differ, expressed as a
	// fraction (0.0-1.0) of the destination edge's length that corresponds
	// to the start (top/left) of the source edge. 0.0 aligns the starting
	// corners (see specs/screen-layout: "Offset de alinhamento").
	Offset float64 `json:"offset"`
}

// Layout is the full screen-adjacency graph configured by the user.
type Layout struct {
	mu    sync.RWMutex
	nodes map[string]ScreenRect
	links map[string]Link // keyed by FromNode+"|"+FromEdge, one outgoing link per node edge
}

// NewLayout creates an empty layout.
func NewLayout() *Layout {
	return &Layout{
		nodes: make(map[string]ScreenRect),
		links: make(map[string]Link),
	}
}

func linkKey(node string, edge Edge) string {
	return node + "|" + string(edge)
}

// SetNode registers or updates a node's logical screen rectangle.
func (l *Layout) SetNode(nodeID string, rect ScreenRect) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nodes[nodeID] = rect
}

// RemoveNode drops a node and every link that touches it, in either direction.
func (l *Layout) RemoveNode(nodeID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.nodes, nodeID)
	for k, link := range l.links {
		if link.FromNode == nodeID || link.ToNode == nodeID {
			delete(l.links, k)
		}
	}
}

// SetLink validates and stores a border adjacency, replacing any existing
// link that starts at the same FromNode/FromEdge (a node edge can only lead
// to one neighbor at a time).
func (l *Layout) SetLink(link Link) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !link.FromEdge.valid() || !link.ToEdge.valid() {
		return fmt.Errorf("inputshare: invalid edge in link %+v", link)
	}
	if link.FromNode == "" || link.ToNode == "" {
		return fmt.Errorf("inputshare: link requires both FromNode and ToNode")
	}
	if link.FromNode == link.ToNode {
		return fmt.Errorf("inputshare: a node cannot link to itself (%s)", link.FromNode)
	}
	if _, ok := l.nodes[link.FromNode]; !ok {
		return fmt.Errorf("inputshare: unknown node %q referenced by link", link.FromNode)
	}
	if _, ok := l.nodes[link.ToNode]; !ok {
		return fmt.Errorf("inputshare: unknown node %q referenced by link", link.ToNode)
	}
	if link.Offset < 0 || link.Offset > 1 {
		return fmt.Errorf("inputshare: link offset must be within [0,1], got %f", link.Offset)
	}

	l.links[linkKey(link.FromNode, link.FromEdge)] = link
	return nil
}

// RemoveLink deletes the outgoing link from a given node edge, if any.
func (l *Layout) RemoveLink(nodeID string, edge Edge) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.links, linkKey(nodeID, edge))
}

// Nodes returns a snapshot of every registered node rectangle, keyed by
// node (device) ID.
func (l *Layout) Nodes() map[string]ScreenRect {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]ScreenRect, len(l.nodes))
	for k, v := range l.nodes {
		out[k] = v
	}
	return out
}

// Links returns a snapshot of every configured link.
func (l *Layout) Links() []Link {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Link, 0, len(l.links))
	for _, link := range l.links {
		out = append(out, link)
	}
	return out
}

// Crossing is the result of a cursor crossing a screen border: the
// destination node and the position it should reappear at, already scaled
// into the destination's coordinate space.
type Crossing struct {
	ToNode string
	ToEdge Edge
	// AlongPx is the position along ToEdge (Y for left/right edges, X for
	// top/bottom edges) where the cursor reappears, in destination pixels.
	AlongPx int
}

// Cross computes what happens when the cursor, currently owned by fromNode,
// reaches fromEdge at the given position along that edge (alongPx, in
// fromNode's own pixel space). It returns ok=false when that edge has no
// configured neighbor, meaning the cursor should be clamped instead of
// transitioning (specs/screen-layout: "Cursor atinge borda sem adjacência
// configurada").
func (l *Layout) Cross(fromNode string, fromEdge Edge, alongPx int) (Crossing, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	link, ok := l.links[linkKey(fromNode, fromEdge)]
	if !ok {
		return Crossing{}, false
	}

	fromRect, ok := l.nodes[fromNode]
	if !ok {
		return Crossing{}, false
	}
	toRect, ok := l.nodes[link.ToNode]
	if !ok {
		return Crossing{}, false
	}

	fromDim := edgeLengthPx(fromRect, fromEdge)
	toDim := edgeLengthPx(toRect, link.ToEdge)
	if fromDim <= 0 || toDim <= 0 {
		return Crossing{}, false
	}

	// Proportional scaling (design.md Decision 5), shifted by the
	// configured alignment offset (design.md Decision 6 / spec "Offset de
	// alinhamento") rather than a plain clamp.
	fraction := float64(alongPx) / float64(fromDim)
	targetAlong := int((link.Offset + fraction) * float64(toDim))
	if targetAlong < 0 {
		targetAlong = 0
	}
	if targetAlong >= toDim {
		targetAlong = toDim - 1
	}

	return Crossing{ToNode: link.ToNode, ToEdge: link.ToEdge, AlongPx: targetAlong}, true
}

// edgeLengthPx returns the length, in pixels, of the dimension that runs
// along the given edge (height for left/right, width for top/bottom).
func edgeLengthPx(rect ScreenRect, edge Edge) int {
	switch edge {
	case EdgeLeft, EdgeRight:
		return rect.HeightPx
	case EdgeTop, EdgeBottom:
		return rect.WidthPx
	default:
		return 0
	}
}

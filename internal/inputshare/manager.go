package inputshare

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"omnidesk/internal/config"
)

// Corner identifies one of the four screen corners for the hot-corner
// escape mechanism (design.md: "hot corner fixo de retorno").
type Corner string

const (
	CornerNone        Corner = ""
	CornerTopLeft     Corner = "top-left"
	CornerTopRight    Corner = "top-right"
	CornerBottomLeft  Corner = "bottom-left"
	CornerBottomRight Corner = "bottom-right"
)

// cornerToleragePx is how close to the literal corner pixel the cursor must
// get to count as "hit the corner" — a single-pixel target would be nearly
// impossible to land on with real mouse hardware.
const cornerTolerancePx = 4

// PeerResolver resolves a paired peer's current network address. The core
// package's Node (mDNS discovery + trusted device list) implements this.
type PeerResolver interface {
	ResolveAddr(peerID string) (addr string, ok bool)
}

// activeSession describes the one input-sharing session this node may be
// part of at a time (design.md Decision 6: ownership is local to a node,
// not globally coordinated — there is only ever one physical mouse in use).
type activeSession struct {
	peerID string
	r      role
	conn   *Conn
}

// Manager wires together the layout graph, the permission model, the
// platform capture/injection backend and the transport into the KVM-style
// input-sharing feature described by openspec/changes/kvm-input-sharing.
type Manager struct {
	mu sync.Mutex

	cfg      *config.Config
	resolver PeerResolver
	perm     *PermissionManager
	layout   *Layout
	backend  Backend

	localNodeID string
	localRect   ScreenRect

	active *activeSession

	// hotkeyCombo is the configured "always return to local" chord; heldSet
	// tracks which of its keys are currently down while a session is active.
	hotkeyCombo []HIDUsage
	heldSet     map[HIDUsage]bool
	hotCorner   Corner

	// cursorX/cursorY mirror the position this node believes the (locally
	// injected) cursor is at while receiving from a peer, used only for
	// hot-corner detection.
	cursorX, cursorY int

	// pausedPeers holds devices for which input sharing is temporarily
	// suspended via the tray/dashboard toggle (specs/input-sharing-transport:
	// "Pausa de compartilhamento de input via bandeja/dashboard") — checked
	// on both the sending (border-crossing) and receiving (accepting) paths
	// so a pause is symmetric regardless of which side initiates.
	pausedPeers map[string]bool

	// unavailableReason explains why the platform capture/inject backend
	// could not start (e.g. a missing macOS Accessibility/Input Monitoring
	// grant), so the dashboard/tray can show an actionable message instead
	// of input-sharing silently doing nothing (specs
	// input-capture-inject-macos: "Onboarding de permissões").
	unavailableReason string
}

// NewManager constructs a Manager. Start must be called before it does
// anything.
func NewManager(cfg *config.Config, resolver PeerResolver) *Manager {
	ishare := cfg.GetInputShareConfig()

	layout := NewLayout()
	for id, n := range ishare.Nodes {
		layout.SetNode(id, ScreenRect{WidthPx: n.WidthPx, HeightPx: n.HeightPx})
	}
	for _, l := range ishare.Links {
		_ = layout.SetLink(Link{
			FromNode: l.FromNode,
			FromEdge: Edge(l.FromEdge),
			ToNode:   l.ToNode,
			ToEdge:   Edge(l.ToEdge),
			Offset:   l.Offset,
		})
	}

	hotkey := make([]HIDUsage, len(ishare.HotkeyHID))
	for i, v := range ishare.HotkeyHID {
		hotkey[i] = HIDUsage(v)
	}

	return &Manager{
		cfg:         cfg,
		resolver:    resolver,
		perm:        NewPermissionManager(cfg),
		layout:      layout,
		localNodeID: cfg.DeviceID,
		hotkeyCombo: hotkey,
		heldSet:     make(map[HIDUsage]bool),
		hotCorner:   Corner(ishare.HotCorner),
		pausedPeers: make(map[string]bool),
	}
}

// Permissions exposes the permission manager for the dashboard/API layer.
func (m *Manager) Permissions() *PermissionManager { return m.perm }

// Layout exposes the layout graph for the dashboard/API layer.
func (m *Manager) Layout() *Layout { return m.layout }

// LocalNodeID is this node's own device ID, as used in the layout graph.
func (m *Manager) LocalNodeID() string { return m.localNodeID }

// PausePeer suspends input sharing with a specific device until ResumePeer
// is called, ending any session with it that is currently active.
func (m *Manager) PausePeer(peerID string) {
	m.mu.Lock()
	m.pausedPeers[peerID] = true
	active := m.active
	m.mu.Unlock()

	if active != nil && active.peerID == peerID {
		m.StopSession()
	}
}

// ResumePeer lifts a pause set by PausePeer.
func (m *Manager) ResumePeer(peerID string) {
	m.mu.Lock()
	delete(m.pausedPeers, peerID)
	m.mu.Unlock()
}

func (m *Manager) isPaused(peerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pausedPeers[peerID]
}

// Settings returns a snapshot of everything the dashboard needs to render
// and edit the screen layout and escape mechanisms in one call.
func (m *Manager) Settings() (nodes map[string]ScreenRect, links []Link, hotkey []HIDUsage, corner Corner) {
	m.mu.Lock()
	hotkey = append([]HIDUsage(nil), m.hotkeyCombo...)
	corner = m.hotCorner
	m.mu.Unlock()
	return m.layout.Nodes(), m.layout.Links(), hotkey, corner
}

// ActiveSession reports the peer currently involved in an input-sharing
// session with this node, if any (task 8.3: "indicador de qual nó está
// atualmente em posse do input").
func (m *Manager) ActiveSession() (peerID string, sending bool, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return "", false, false
	}
	return m.active.peerID, m.active.r == roleSender, true
}

// Start initializes the platform capture/injection backend and begins
// watching local input for border crossings and the hotkey escape. If no
// backend exists for this platform/session (e.g. Wayland today), input
// sharing is disabled but the rest of OmniDesk keeps working — mirroring
// how clipboard degrades gracefully when running headless.
func (m *Manager) Start(ctx context.Context) error {
	backend, err := NewBackend()
	if err != nil {
		m.setUnavailable(err)
		log.Printf("[inputshare] desabilitado: %v", err)
		return nil
	}
	m.backend = backend

	rect, err := backend.ScreenRect()
	if err != nil {
		m.setUnavailable(err)
		log.Printf("[inputshare] desabilitado: falha ao ler geometria da tela: %v", err)
		return nil
	}
	m.localRect = rect
	m.layout.SetNode(m.localNodeID, rect)

	cb := Callbacks{
		OnMotion: m.handleLocalMotion,
		OnButton: m.handleLocalButton,
		OnScroll: m.handleLocalScroll,
		OnKey:    m.handleLocalKey,
	}
	if err := backend.Start(ctx, cb); err != nil {
		wrapped := fmt.Errorf("inputshare: failed to start %s capture: %w", backend.Name(), err)
		m.setUnavailable(wrapped)
		return wrapped
	}

	log.Printf("[inputshare] ativo (%s), tela local %dx%d", backend.Name(), rect.WidthPx, rect.HeightPx)
	return nil
}

func (m *Manager) setUnavailable(err error) {
	m.mu.Lock()
	m.unavailableReason = err.Error()
	m.mu.Unlock()
}

// UnavailableReason reports why input-sharing failed to start on this node,
// if it did, so the dashboard/tray can surface it instead of only the log
// (specs input-capture-inject-macos: "Onboarding de permissões... em vez de
// falhar silenciosamente").
func (m *Manager) UnavailableReason() (reason string, unavailable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.unavailableReason, m.unavailableReason != ""
}

// Stop ends any active session and releases the capture backend.
func (m *Manager) Stop() {
	m.StopSession()
	if m.backend != nil {
		m.backend.Stop()
	}
}

// SetHotkey updates and persists the "always return to local" key chord.
func (m *Manager) SetHotkey(combo []HIDUsage) error {
	m.mu.Lock()
	m.hotkeyCombo = combo
	m.mu.Unlock()
	return m.persistLayout()
}

// SetHotCorner updates and persists the reserved return corner.
func (m *Manager) SetHotCorner(c Corner) error {
	m.mu.Lock()
	m.hotCorner = c
	m.mu.Unlock()
	return m.persistLayout()
}

// LocalRect returns this node's live desktop screen dimensions.
func (m *Manager) LocalRect() ScreenRect {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.localRect
}

// UpdatePeerResolution dynamically registers or updates a peer's resolution
// as discovered on the LAN or established via session handshake.
func (m *Manager) UpdatePeerResolution(peerID string, width, height int) {
	if width <= 0 || height <= 0 || peerID == "" || peerID == m.localNodeID {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	current, exists := m.layout.Nodes()[peerID]
	if !exists || current.WidthPx != width || current.HeightPx != height {
		m.layout.SetNode(peerID, ScreenRect{WidthPx: width, HeightPx: height})
		_ = m.persistLayout()
	}
}

// SetLayout replaces the screen-arrangement graph and persists it
// (task 4.4: endpoints to read/save the configured layout).
func (m *Manager) SetLayout(nodes map[string]ScreenRect, links []Link) error {
	newLayout := NewLayout()
	for id, r := range nodes {
		newLayout.SetNode(id, r)
	}
	for _, l := range links {
		if err := newLayout.SetLink(l); err != nil {
			return err
		}
	}
	// Always keep this node's own live geometry, even if the caller's
	// snapshot predates it.
	newLayout.SetNode(m.localNodeID, m.localRect)

	m.mu.Lock()
	m.layout = newLayout
	m.mu.Unlock()
	return m.persistLayout()
}

func (m *Manager) persistLayout() error {
	m.mu.Lock()
	nodes := m.layout.Nodes()
	links := m.layout.Links()
	hotkey := make([]int, len(m.hotkeyCombo))
	for i, h := range m.hotkeyCombo {
		hotkey[i] = int(h)
	}
	hotCorner := m.hotCorner
	m.mu.Unlock()

	cfgNodes := make(map[string]config.ScreenNode, len(nodes))
	for id, r := range nodes {
		cfgNodes[id] = config.ScreenNode{WidthPx: r.WidthPx, HeightPx: r.HeightPx}
	}
	cfgLinks := make([]config.ScreenLink, 0, len(links))
	for _, l := range links {
		cfgLinks = append(cfgLinks, config.ScreenLink{
			FromNode: l.FromNode, FromEdge: string(l.FromEdge),
			ToNode: l.ToNode, ToEdge: string(l.ToEdge), Offset: l.Offset,
		})
	}

	return m.cfg.SetInputShareConfig(config.InputShareConfig{
		Nodes: cfgNodes, Links: cfgLinks, HotkeyHID: hotkey, HotCorner: string(hotCorner),
	})
}

// AcceptSession authorizes and starts the receiving side of an
// input-sharing session for an already-authenticated HTTP request. It
// enforces the input-control permission (specs/input-control-permission)
// and the one-session-at-a-time invariant before ever touching the
// WebSocket, so an unauthorized or duplicate request gets a normal HTTP
// error instead of an upgraded-then-closed connection.
func (m *Manager) AcceptSession(w http.ResponseWriter, r *http.Request, peerID string) error {
	if m.backend == nil {
		return fmt.Errorf("inputshare: not available on this platform/session")
	}
	if !m.perm.IsGrantedTo(peerID) {
		return fmt.Errorf("inputshare: peer is not authorized to control this device")
	}
	if m.isPaused(peerID) {
		return fmt.Errorf("inputshare: sharing with this peer is paused")
	}

	m.mu.Lock()
	if m.active != nil {
		m.mu.Unlock()
		return fmt.Errorf("inputshare: a session is already active")
	}
	m.mu.Unlock()

	conn, err := Accept(w, r, peerID, m.localRect)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.active = &activeSession{peerID: peerID, r: roleReceiver, conn: conn}
	m.cursorX, m.cursorY = m.localRect.WidthPx/2, m.localRect.HeightPx/2
	m.mu.Unlock()

	conn.SetReceiverHandlers(m.injectAndTrack, func() { m.clearSession(peerID) })
	conn.Start()
	log.Printf("[inputshare] recebendo controle de %s", peerID)
	return nil
}

// injectAndTrack forwards a received event to the platform backend while
// mirroring the resulting cursor position, so hot-corner and edge-return
// detection work without querying the OS on every event.
func (m *Manager) injectAndTrack(ev Event) error {
	var shouldReturn bool

	m.mu.Lock()
	switch e := ev.(type) {
	case MouseWarpEvent:
		// Clamp entry warp coordinates within local screen boundaries.
		// If the sender had an inaccurate ScreenRect for this node (e.g. default 1920x1080
		// while this display is 1728x1117 or 1512x982), an unclamped entry point can land
		// right on or past the edge, causing an immediate bounce-back.
		const inset = 20
		x := int(e.X)
		y := int(e.Y)
		maxX := m.localRect.WidthPx - 1
		maxY := m.localRect.HeightPx - 1

		if maxX > 2*inset {
			if x > maxX-inset {
				x = maxX - inset
			}
			if x < inset {
				x = inset
			}
		} else if maxX > 0 {
			x = clampInt(x, 0, maxX)
		}

		if maxY > 2*inset {
			if y > maxY-inset {
				y = maxY - inset
			}
			if y < inset {
				y = inset
			}
		} else if maxY > 0 {
			y = clampInt(y, 0, maxY)
		}

		m.cursorX = x
		m.cursorY = y
		ev = MouseWarpEvent{X: uint16(x), Y: uint16(y)}

	case MouseMoveEvent:
		m.cursorX += int(e.DX)
		m.cursorY += int(e.DY)
		if m.localRect.WidthPx > 0 {
			m.cursorX = clampInt(m.cursorX, 0, m.localRect.WidthPx-1)
		}
		if m.localRect.HeightPx > 0 {
			m.cursorY = clampInt(m.cursorY, 0, m.localRect.HeightPx-1)
		}

		if m.active != nil && m.shouldRequestReturn(m.cursorX, m.cursorY, m.hotCorner) {
			shouldReturn = true
		}
	}
	active := m.active
	cx, cy := m.cursorX, m.cursorY
	rect := m.localRect
	m.mu.Unlock()

	if shouldReturn && active != nil {
		log.Printf("[inputshare] tracked cursor (%d,%d) on %dx%d hit return trigger, requesting control back from %s",
			cx, cy, rect.WidthPx, rect.HeightPx, active.peerID)
		active.conn.SendRequestReturn()
	}

	if m.backend != nil {
		return m.backend.Inject(ev)
	}
	return nil
}

// shouldRequestReturn reports whether the tracked cursor of a node
// currently *receiving* control has reached a point that should hand
// ownership back to whoever is sending: either the configured hot corner,
// or — per specs/screen-layout's "Transição de borda dispara mudança de
// posse de input", which is written generally, not scoped to "only when
// idle" — any other screen edge that has a configured neighbor. Without
// this, a node being controlled has no way to hand input back by crossing
// out the way it came in; the only escapes left are the hotkey and a
// manual dashboard/tray pause, which is only reachable if a local browser
// isn't ALSO grabbed away by Suppress on the sender (see tasks.md 7.7/7.8).
//
// This always routes back to the current sender rather than relaying to a
// third node even when the crossed edge's configured neighbor is someone
// else — genuine multi-hop handoff is future work (tasks.md 7.8); "give
// control back to whoever has it" is the safe behavior for every edge in
// the meantime.
func (m *Manager) shouldRequestReturn(x, y int, corner Corner) bool {
	if corner != CornerNone && hitsCorner(x, y, m.localRect, corner) {
		return true
	}
	edge, along, atEdge := m.edgeAt(x, y)
	if !atEdge {
		return false
	}
	_, ok := m.layout.Cross(m.localNodeID, edge, along)
	return ok
}

func hitsCorner(x, y int, rect ScreenRect, c Corner) bool {
	near := func(v, edge int) bool {
		d := v - edge
		if d < 0 {
			d = -d
		}
		return d <= cornerTolerancePx
	}
	switch c {
	case CornerTopLeft:
		return near(x, 0) && near(y, 0)
	case CornerTopRight:
		return near(x, rect.WidthPx-1) && near(y, 0)
	case CornerBottomLeft:
		return near(x, 0) && near(y, rect.HeightPx-1)
	case CornerBottomRight:
		return near(x, rect.WidthPx-1) && near(y, rect.HeightPx-1)
	default:
		return false
	}
}

// RequestControlOf asks a paired peer to grant this node permission to
// control it, over the normal authenticated peer-to-peer HTTP API (not the
// input WebSocket). The peer's dashboard surfaces the request for a human
// to approve (specs/input-control-permission: "Concessão de permissão exige
// confirmação explícita nas duas pontas").
func (m *Manager) RequestControlOf(ctx context.Context, peerID string) error {
	addr, ok := m.resolver.ResolveAddr(peerID)
	if !ok {
		return fmt.Errorf("inputshare: peer %s is not currently reachable", peerID)
	}
	dev, ok := m.cfg.GetTrustedDevice(peerID)
	if !ok {
		return fmt.Errorf("inputshare: peer %s is not paired", peerID)
	}
	return sendPermissionRequest(ctx, addr, m.cfg.DeviceID, m.cfg.DeviceName, dev.Token)
}

// StopSession ends whatever session is currently active, regardless of
// role. As the sender, it proactively tells the peer to release everything
// before closing (design.md Decision 7's "best effort" half — the
// receiver's self-sufficient failsafe covers the rest). This is the
// implementation behind all three escape mechanisms (hotkey, hot corner,
// bandeja/dashboard toggle) and behind a clean edge-triggered handoff back.
func (m *Manager) StopSession() {
	m.mu.Lock()
	active := m.active
	m.active = nil
	m.heldSet = make(map[HIDUsage]bool)
	m.mu.Unlock()

	if active == nil {
		return
	}
	log.Printf("[inputshare] StopSession triggered for %s (role=%v)", active.peerID, active.r)
	if active.r == roleSender {
		active.conn.SendReleaseAll()
		m.backend.Release()
	}
	active.conn.Close()
	log.Printf("[inputshare] sessão com %s encerrada", active.peerID)
}

// clearSession ends a receive session from the receiving side's own
// bookkeeping (called when the sender closes the connection — including
// right after this same node asked for control back, see
// shouldRequestReturn). It only ever runs for role receiver: the sender
// side tears down through StopSession instead.
//
// Ending a receive session always leaves the OS-level cursor sitting
// exactly at whatever edge triggered the handoff — that's the position
// that made shouldRequestReturn fire in the first place. The moment
// m.active goes nil, this node's own capture re-arms edge detection
// (handleLocalMotion), and it's now watching a cursor parked right on the
// trigger line: the next motion sample from *anything* — a fraction of a
// pixel of jitter, unrelated local X activity, even a stray leftover
// event from the session that just ended — reads as "still at the edge,
// still trying to leave" and starts a brand new session straight back
// out, with roles swapped from what either side expects. Warping to the
// screen's center gives a real margin against that before re-arming.
func (m *Manager) clearSession(peerID string) {
	m.mu.Lock()
	cleared := m.active != nil && m.active.peerID == peerID
	cx, cy := m.cursorX, m.cursorY
	if cleared {
		m.active = nil
		m.heldSet = make(map[HIDUsage]bool)
	}
	rect := m.localRect
	m.mu.Unlock()

	if cleared {
		log.Printf("[inputshare] receive session with %s ended (tracked cursor was at %d,%d on %dx%d), recentering",
			peerID, cx, cy, rect.WidthPx, rect.HeightPx)
	}

	if cleared && m.backend != nil {
		m.backend.Inject(MouseWarpEvent{X: uint16(rect.WidthPx / 2), Y: uint16(rect.HeightPx / 2)})
	}
}

// handleLocalMotion is the capture backend's pointer-motion callback. While
// idle it watches for the cursor reaching a screen edge with a configured
// neighbor; while sending, it forwards motion to the peer.
func (m *Manager) handleLocalMotion(absX, absY int, dx, dy int16) {
	m.mu.Lock()
	active := m.active
	m.mu.Unlock()

	if active != nil {
		if active.r == roleSender {
			active.conn.SendMove(dx, dy)
		}
		return
	}

	edge, along, atEdge := m.edgeAt(absX, absY)
	if !atEdge {
		return
	}
	m.tryBeginSending(edge, along)
}

// edgeAt reports which screen edge (absX, absY) is touching, if any, along
// with the coordinate that runs along that edge.
func (m *Manager) edgeAt(absX, absY int) (edge Edge, along int, ok bool) {
	w, h := m.localRect.WidthPx, m.localRect.HeightPx
	switch {
	case absX <= 0:
		return EdgeLeft, absY, true
	case absX >= w-1:
		return EdgeRight, absY, true
	case absY <= 0:
		return EdgeTop, absX, true
	case absY >= h-1:
		return EdgeBottom, absX, true
	default:
		return "", 0, false
	}
}

// tryBeginSending looks up the configured neighbor for the crossed edge and,
// if this node is authorized to control it, dials out and becomes the
// session's sender (specs/screen-layout: "Transição de borda dispara
// mudança de posse de input").
func (m *Manager) tryBeginSending(edge Edge, along int) {
	crossing, ok := m.layout.Cross(m.localNodeID, edge, along)
	if !ok {
		return // no neighbor configured on this edge — nothing to do, OS clamps the cursor on its own
	}
	peerID := crossing.ToNode
	if m.isPaused(peerID) {
		return
	}

	addr, ok := m.resolver.ResolveAddr(peerID)
	if !ok {
		return // peer offline; stay local
	}
	dev, ok := m.cfg.GetTrustedDevice(peerID)
	if !ok || dev.Token == "" {
		return
	}

	ctx := context.Background()
	conn, peerRect, err := DialSender(ctx, addr, m.cfg.DeviceID, dev.Token, peerID)
	if err != nil {
		log.Printf("[inputshare] não foi possível iniciar controle de %s: %v", peerID, err)
		return
	}

	dstRect := m.layout.Nodes()[peerID]
	if peerRect.WidthPx > 0 && peerRect.HeightPx > 0 {
		dstRect = peerRect
		m.mu.Lock()
		m.layout.SetNode(peerID, peerRect)
		m.mu.Unlock()
		_ = m.persistLayout()
	}

	entryX, entryY := warpEntryPoint(crossing, dstRect)

	// Suppress before publishing m.active: once handleLocalMotion sees an
	// active sender session it starts forwarding instead of watching for
	// edges, so the grab must already be in place by then — otherwise a
	// window between "active" and "grabbed" would leak local delivery of
	// whatever the user does in that gap.
	if err := m.backend.Suppress(); err != nil {
		log.Printf("[inputshare] não foi possível suprimir input local, abortando controle de %s: %v", peerID, err)
		conn.Close()
		return
	}

	m.mu.Lock()
	if m.active != nil {
		// Lost a race with an incoming session; abandon this attempt.
		m.mu.Unlock()
		conn.Close()
		m.backend.Release()
		return
	}
	m.active = &activeSession{peerID: peerID, r: roleSender, conn: conn}
	m.heldSet = make(map[HIDUsage]bool)
	m.mu.Unlock()

	conn.SetSenderHandlers(func() { m.StopSession() })
	conn.Start()
	conn.SendWarp(entryX, entryY)
	log.Printf("[inputshare] controlando %s", peerID)
}

// warpEntryPoint turns a Crossing (which already carries the scaled
// along-edge coordinate) into a full (X, Y) pair in the destination's pixel
// space, placing the fixed coordinate just inside the entry edge.
func warpEntryPoint(c Crossing, dst ScreenRect) (x, y uint16) {
	// 1px put the entry point right next to the edge-return threshold
	// shouldRequestReturn checks (tasks.md 7.8/7.9): any 1px of jitter
	// right after crossing in immediately bounced control back out. A
	// real margin means an accidental wobble doesn't read as "didn't
	// migrate" — bouncing back out now takes a deliberate move.
	const inset = 20
	switch c.ToEdge {
	case EdgeLeft:
		return inset, uint16(clampInt(c.AlongPx, 0, dst.HeightPx-1))
	case EdgeRight:
		return uint16(dst.WidthPx - 1 - inset), uint16(clampInt(c.AlongPx, 0, dst.HeightPx-1))
	case EdgeTop:
		return uint16(clampInt(c.AlongPx, 0, dst.WidthPx-1)), inset
	case EdgeBottom:
		return uint16(clampInt(c.AlongPx, 0, dst.WidthPx-1)), uint16(dst.HeightPx - 1 - inset)
	default:
		return 0, 0
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m *Manager) handleLocalButton(btn MouseButton, pressed bool) {
	m.mu.Lock()
	active := m.active
	m.mu.Unlock()
	if active != nil && active.r == roleSender {
		active.conn.SendButton(btn, pressed)
	}
}

func (m *Manager) handleLocalScroll(dx, dy int16) {
	m.mu.Lock()
	active := m.active
	m.mu.Unlock()
	if active != nil && active.r == roleSender {
		active.conn.SendScroll(dx, dy)
	}
}

// panicHotkeyCombo is a fixed, non-configurable "give me my computer back"
// chord, checked in addition to whatever the user configured as their own
// return hotkey (which may be unset). Suppress() grabs the pointer and
// keyboard on the sender for the duration of a session (tasks.md 7.7),
// which also blocks local interaction with this same machine's own
// dashboard/tray pause button — so unlike the other two escapes, this one
// cannot depend on any prior setup existing. Four modifiers held together
// is deliberately unlikely to be pressed by accident or bound by a window
// manager.
var panicHotkeyCombo = []HIDUsage{
	HIDKeyLeftControl, HIDKeyLeftAlt, HIDKeyLeftShift, HIDKeyEscape,
}

// handleLocalKey forwards key events while sending, and — before
// forwarding — checks whether the physical origin's hotkey chord (the
// user-configured one, or the always-on panicHotkeyCombo) is now fully
// held, in which case it swallows the combo and reclaims local control
// instead of forwarding it (design.md: "hotkey global de retorno").
func (m *Manager) handleLocalKey(hid HIDUsage, pressed bool) {
	m.mu.Lock()
	active := m.active
	if active == nil || active.r != roleSender {
		m.mu.Unlock()
		return
	}

	if pressed {
		m.heldSet[hid] = true
	} else {
		delete(m.heldSet, hid)
	}
	hotkeyMatched := (len(m.hotkeyCombo) > 0 && allHeld(m.heldSet, m.hotkeyCombo)) ||
		allHeld(m.heldSet, panicHotkeyCombo)
	m.mu.Unlock()

	if hotkeyMatched {
		m.StopSession()
		return
	}
	active.conn.SendKey(hid, pressed)
}

func allHeld(held map[HIDUsage]bool, combo []HIDUsage) bool {
	for _, hid := range combo {
		if !held[hid] {
			return false
		}
	}
	return true
}

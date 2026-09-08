package inputshare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"omnidesk/internal/config"
)

// PermissionRequestPayload is the body of an outbound request asking a peer
// to grant this node permission to control it (POST
// /api/v1/input/permission/request, authenticated like every other
// peer-to-peer OmniDesk endpoint).
type PermissionRequestPayload struct {
	RequesterName string `json:"requester_name"`
}

// sendPermissionRequest performs the outbound call. It reuses the same
// device-token header scheme as clipboard/file-transfer requests
// (Server.authMiddleware on the receiving end).
func sendPermissionRequest(ctx context.Context, addr, deviceID, deviceName, token string) error {
	body, err := json.Marshal(PermissionRequestPayload{RequesterName: deviceName})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/api/v1/input/permission/request", addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OmniDesk-Device-ID", deviceID)
	req.Header.Set("X-OmniDesk-Token", token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("inputshare: peer rejected permission request (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// PendingRequest is an incoming, not-yet-decided request from a paired peer
// asking to be granted permission to control this node's mouse/keyboard.
// Like pairing.Session, it lives only in memory: once approved it becomes
// durable state on the TrustedDevice record; once denied or expired it
// simply disappears (specs/input-control-permission: "Solicitação pendente
// até aprovação").
type PendingRequest struct {
	PeerID      string    `json:"peer_id"`
	PeerName    string    `json:"peer_name"`
	RequestedAt time.Time `json:"requested_at"`
}

const pendingRequestTTL = 5 * time.Minute

// PermissionManager owns the opt-in, per-direction, per-peer permission to
// control this node's input (design.md Decision 4) — a concern kept
// deliberately separate from device pairing (internal/pairing), which only
// establishes the mutually-trusted token used to authenticate requests.
type PermissionManager struct {
	mu      sync.Mutex
	cfg     *config.Config
	pending map[string]PendingRequest // keyed by PeerID
}

// NewPermissionManager creates a manager backed by the node's persisted
// trusted-device list.
func NewPermissionManager(cfg *config.Config) *PermissionManager {
	return &PermissionManager{
		cfg:     cfg,
		pending: make(map[string]PendingRequest),
	}
}

// IsGrantedTo reports whether peerID is currently allowed to control this
// node's input. This is the single enforcement check consulted before
// accepting an input-sharing WebSocket (specs/input-control-permission).
func (p *PermissionManager) IsGrantedTo(peerID string) bool {
	dev, ok := p.cfg.GetTrustedDevice(peerID)
	if !ok {
		return false
	}
	return dev.InputControlGranted
}

// RegisterRequest records an incoming request from a paired peer. It does
// not grant anything by itself — a user must call Approve.
func (p *PermissionManager) RegisterRequest(peerID, peerName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[peerID] = PendingRequest{PeerID: peerID, PeerName: peerName, RequestedAt: time.Now()}
}

// Pending returns every not-yet-expired pending request.
func (p *PermissionManager) Pending() []PendingRequest {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]PendingRequest, 0, len(p.pending))
	now := time.Now()
	for id, req := range p.pending {
		if now.Sub(req.RequestedAt) > pendingRequestTTL {
			delete(p.pending, id)
			continue
		}
		out = append(out, req)
	}
	return out
}

// Approve grants peerID permission to control this node's input, persisting
// the decision on its TrustedDevice record so it survives a restart and is
// automatically revoked if the device is ever unpaired
// (specs/input-control-permission: "Revogação em cascata").
func (p *PermissionManager) Approve(peerID string) error {
	p.mu.Lock()
	delete(p.pending, peerID)
	p.mu.Unlock()

	dev, ok := p.cfg.GetTrustedDevice(peerID)
	if !ok {
		return fmt.Errorf("inputshare: cannot grant input control to unpaired device %q", peerID)
	}
	dev.InputControlGranted = true
	return p.cfg.AddTrustedDevice(dev)
}

// Deny discards a pending request without granting anything.
func (p *PermissionManager) Deny(peerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, peerID)
}

// Revoke removes a previously granted permission without unpairing the
// device (specs/input-control-permission: "Revogação manual independente do
// despareamento").
func (p *PermissionManager) Revoke(peerID string) error {
	dev, ok := p.cfg.GetTrustedDevice(peerID)
	if !ok {
		return nil
	}
	if !dev.InputControlGranted {
		return nil
	}
	dev.InputControlGranted = false
	return p.cfg.AddTrustedDevice(dev)
}

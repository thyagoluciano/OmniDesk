package clipboard

import (
	"testing"
	"time"

	"omnidesk/internal/config"
	"omnidesk/internal/discovery"
)

type mockPeerProvider struct {
	peers []discovery.DiscoveredPeer
}

func (m *mockPeerProvider) GetOnlineTrustedPeers() []discovery.DiscoveredPeer {
	return m.peers
}

func TestAntiEchoMechanism(t *testing.T) {
	cfg := &config.Config{
		DeviceID:      "node-local",
		DeviceName:    "MacBook",
		ClipboardSync: true,
	}

	peers := &mockPeerProvider{}
	engine := NewEngine(cfg, peers)

	text := "Confidential test token"
	h := HashText(text)

	// Step 1: Inject remote clipboard
	err := engine.InjectRemoteClipboard(text, "remote-peer")
	if err != nil {
		t.Fatalf("InjectRemoteClipboard failed: %v", err)
	}

	// Verify hash is recorded in anti-echo table
	engine.mu.RLock()
	exp, exists := engine.antiEcho[h]
	engine.mu.RUnlock()

	if !exists {
		t.Fatalf("expected hash to be recorded in anti-echo cache")
	}
	if time.Now().After(exp) {
		t.Fatalf("anti-echo expiration should be in the future")
	}

	// Step 2: Simulate local clipboard event with the exact same text
	// HandleLocalCopy should recognize anti-echo and NOT broadcast
	engine.HandleLocalCopy(text)

	// Step 3: Test pause sync
	cfg.ClipboardSync = false
	if engine.IsSyncEnabled() {
		t.Errorf("expected sync to be disabled")
	}

	err = engine.InjectRemoteClipboard("new text", "remote-peer")
	if err != nil {
		t.Fatalf("inject should succeed without error even when paused")
	}
	// Verify that new text was NOT recorded in anti-echo when paused
	h2 := HashText("new text")
	engine.mu.RLock()
	_, exists2 := engine.antiEcho[h2]
	engine.mu.RUnlock()
	if exists2 {
		t.Errorf("when paused, inject should not register hash")
	}
}

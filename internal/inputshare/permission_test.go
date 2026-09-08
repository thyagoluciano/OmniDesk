package inputshare

import (
	"testing"
	"time"

	"omnidesk/internal/config"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	return cfg
}

func TestPermissionGrantIsSeparateFromPairing(t *testing.T) {
	cfg := newTestConfig(t)
	peer := config.TrustedDevice{ID: "peer-1", Name: "Laptop", Token: "shared-secret", AddedAt: time.Now()}
	if err := cfg.AddTrustedDevice(peer); err != nil {
		t.Fatalf("AddTrustedDevice failed: %v", err)
	}

	perm := NewPermissionManager(cfg)

	// specs/input-control-permission: "Pareamento não concede controle de input".
	if perm.IsGrantedTo("peer-1") {
		t.Fatal("pairing alone must not grant input control")
	}

	perm.RegisterRequest("peer-1", "Laptop")
	pending := perm.Pending()
	if len(pending) != 1 || pending[0].PeerID != "peer-1" {
		t.Fatalf("expected one pending request for peer-1, got %+v", pending)
	}

	if err := perm.Approve("peer-1"); err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if !perm.IsGrantedTo("peer-1") {
		t.Error("expected permission to be granted after approval")
	}
	if len(perm.Pending()) != 0 {
		t.Error("expected no pending requests after approval")
	}
}

func TestPermissionRevokeWithoutUnpairing(t *testing.T) {
	cfg := newTestConfig(t)
	peer := config.TrustedDevice{ID: "peer-2", Name: "Desktop", Token: "shared-secret", InputControlGranted: true}
	if err := cfg.AddTrustedDevice(peer); err != nil {
		t.Fatalf("AddTrustedDevice failed: %v", err)
	}

	perm := NewPermissionManager(cfg)
	if !perm.IsGrantedTo("peer-2") {
		t.Fatal("expected permission granted from fixture")
	}

	if err := perm.Revoke("peer-2"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	if perm.IsGrantedTo("peer-2") {
		t.Error("expected permission revoked")
	}

	// specs/input-control-permission: "Revogar controle sem desparear" — the
	// pairing itself must survive.
	if _, ok := cfg.GetTrustedDevice("peer-2"); !ok {
		t.Error("expected the device to remain paired after revoking input control")
	}
}

func TestPermissionRevokedAutomaticallyOnUnpair(t *testing.T) {
	cfg := newTestConfig(t)
	peer := config.TrustedDevice{ID: "peer-3", Name: "MacMini", Token: "shared-secret", InputControlGranted: true}
	if err := cfg.AddTrustedDevice(peer); err != nil {
		t.Fatalf("AddTrustedDevice failed: %v", err)
	}

	perm := NewPermissionManager(cfg)
	if !perm.IsGrantedTo("peer-3") {
		t.Fatal("expected permission granted from fixture")
	}

	if err := cfg.RemoveTrustedDevice("peer-3"); err != nil {
		t.Fatalf("RemoveTrustedDevice failed: %v", err)
	}

	if perm.IsGrantedTo("peer-3") {
		t.Error("expected input control permission to be gone once the device is unpaired")
	}
}

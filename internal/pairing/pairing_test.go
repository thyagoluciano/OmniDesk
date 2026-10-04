package pairing_test

import (
	"testing"

	"omnidesk/internal/config"
	"omnidesk/internal/pairing"
)

func newTestManager(t *testing.T) (*pairing.Manager, *config.Config) {
	t.Helper()
	cfg := &config.Config{
		DeviceID:       "test-desk",
		DeviceName:     "Test Desktop",
		ListenPort:     24850,
		TrustedDevices: make(map[string]config.TrustedDevice),
	}
	return pairing.NewManager(cfg), cfg
}

func TestQRSessionLifecycle(t *testing.T) {
	mgr, cfg := newTestManager(t)

	sess := mgr.CreateQRSession()
	if sess == nil {
		t.Fatal("expected non-nil QR session")
	}
	if len(sess.PIN) != 6 {
		t.Fatalf("expected 6-digit PIN, got %q", sess.PIN)
	}
	if sess.ResponderID != "test-desk" {
		t.Fatalf("expected responder id 'test-desk', got %q", sess.ResponderID)
	}

	// Session should not be approved yet
	_, approved := mgr.IsQRSessionApproved(sess.PIN)
	if approved {
		t.Fatal("session should not be approved before redeem")
	}

	// Redeem with invalid PIN should fail
	_, _, err := mgr.RedeemQRSession("999999", "mob-1", "My iPhone", "192.168.1.50:24851")
	if err == nil {
		t.Fatal("expected error on invalid PIN redeem")
	}

	// Redeem with valid PIN
	redeemedSess, token, err := mgr.RedeemQRSession(sess.PIN, "mob-1", "My iPhone", "192.168.1.50:24851")
	if err != nil {
		t.Fatalf("unexpected error on valid PIN redeem: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if redeemedSess.RequesterID != "mob-1" {
		t.Fatalf("expected requester id 'mob-1', got %q", redeemedSess.RequesterID)
	}

	// Check IsQRSessionApproved
	appSess, approved := mgr.IsQRSessionApproved(sess.PIN)
	if !approved || appSess == nil {
		t.Fatal("expected session to be approved")
	}

	// Check that device is now in config trusted list
	dev, ok := cfg.GetTrustedDevice("mob-1")
	if !ok {
		t.Fatal("expected mob-1 to be in trusted list")
	}
	if dev.Name != "My iPhone" || dev.Token != token {
		t.Fatalf("trusted device mismatch: %+v", dev)
	}
}

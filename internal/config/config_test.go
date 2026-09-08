package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigLoadAndTrustedDevices(t *testing.T) {
	tempDir := t.TempDir()
	cfgFile := filepath.Join(tempDir, "config.json")

	cfg := &Config{
		DeviceID:       "node-1234",
		DeviceName:     "Test-Machine",
		ListenPort:     24850,
		DownloadDir:    tempDir,
		ClipboardSync:  true,
		TrustedDevices: make(map[string]TrustedDevice),
		configPath:     cfgFile,
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	if _, err := os.Stat(cfgFile); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	// Add trusted device
	dev := TrustedDevice{
		ID:       "node-5678",
		Name:     "MacBook-Air",
		Token:    "secret-token-abc",
		AddedAt:  time.Now(),
		LastSeen: time.Now(),
		LastAddr: "192.168.1.100:24850",
	}

	if err := cfg.AddTrustedDevice(dev); err != nil {
		t.Fatalf("failed to add trusted device: %v", err)
	}

	if !cfg.IsTrusted("node-5678", "secret-token-abc") {
		t.Errorf("expected device to be trusted with correct token")
	}

	if cfg.IsTrusted("node-5678", "wrong-token") {
		t.Errorf("expected device to NOT be trusted with wrong token")
	}

	list := cfg.ListTrustedDevices()
	if len(list) != 1 || list[0].Name != "MacBook-Air" {
		t.Errorf("unexpected trusted devices list: %+v", list)
	}

	// Toggle clipboard sync
	if err := cfg.SetClipboardSync(false); err != nil {
		t.Fatalf("failed to set clipboard sync: %v", err)
	}
	if cfg.IsClipboardSyncEnabled() {
		t.Errorf("expected clipboard sync to be false")
	}
}

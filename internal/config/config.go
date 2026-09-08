package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TrustedDevice represents a peer device paired via PIN.
type TrustedDevice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token"` // Shared mutual authentication token
	AddedAt   time.Time `json:"added_at"`
	LastSeen  time.Time `json:"last_seen"`
	LastAddr  string    `json:"last_addr"`
}

// Config stores local node configuration and trusted peers.
type Config struct {
	mu             sync.RWMutex
	DeviceID       string                   `json:"device_id"`
	DeviceName     string                   `json:"device_name"`
	ListenPort     int                      `json:"listen_port"`
	DownloadDir    string                   `json:"download_dir"`
	ClipboardSync  bool                     `json:"clipboard_sync"`
	TrustedDevices map[string]TrustedDevice `json:"trusted_devices"` // key is DeviceID
	configPath     string
}

// DefaultConfigDir returns the default directory for omnidesk configuration.
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "omnidesk")
}

// DefaultDownloadDir returns the default inbound folder.
func DefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Downloads", "OmniDesk")
}

// Load loads the configuration from disk, creating default values if missing.
func Load() (*Config, error) {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	downloadDir := DefaultDownloadDir()
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download dir: %w", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	cfg := &Config{
		ListenPort:     24850,
		DownloadDir:    downloadDir,
		ClipboardSync:  true,
		TrustedDevices: make(map[string]TrustedDevice),
		configPath:     cfgPath,
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "OmniDesk-Node"
	}
	cfg.DeviceName = hostname

	data, err := os.ReadFile(cfgPath)
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		if cfg.TrustedDevices == nil {
			cfg.TrustedDevices = make(map[string]TrustedDevice)
		}
	} else if os.IsNotExist(err) {
		// Generate unique DeviceID
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		cfg.DeviceID = hex.EncodeToString(b)
		_ = cfg.Save()
	} else {
		return nil, err
	}

	return cfg, nil
}

// Save persists the configuration to disk.
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.configPath == "" {
		return nil
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configPath, data, 0600)
}

// AddTrustedDevice stores a newly paired device.
func (c *Config) AddTrustedDevice(device TrustedDevice) error {
	c.mu.Lock()
	if c.TrustedDevices == nil {
		c.TrustedDevices = make(map[string]TrustedDevice)
	}
	c.TrustedDevices[device.ID] = device
	c.mu.Unlock()
	return c.Save()
}

// RemoveTrustedDevice removes a device from the trusted list.
func (c *Config) RemoveTrustedDevice(deviceID string) error {
	c.mu.Lock()
	delete(c.TrustedDevices, deviceID)
	c.mu.Unlock()
	return c.Save()
}

// IsTrusted validates if a device ID and token are valid and trusted.
func (c *Config) IsTrusted(deviceID, token string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	dev, ok := c.TrustedDevices[deviceID]
	if !ok {
		return false
	}
	return dev.Token == token && token != ""
}

// GetTrustedDevice returns a copy of a trusted device entry if found.
func (c *Config) GetTrustedDevice(deviceID string) (TrustedDevice, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	dev, ok := c.TrustedDevices[deviceID]
	return dev, ok
}

// ListTrustedDevices returns a slice of all currently trusted devices.
func (c *Config) ListTrustedDevices() []TrustedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()
	list := make([]TrustedDevice, 0, len(c.TrustedDevices))
	for _, dev := range c.TrustedDevices {
		list = append(list, dev)
	}
	return list
}

// SetClipboardSync toggles clipboard synchronization.
func (c *Config) SetClipboardSync(enabled bool) error {
	c.mu.Lock()
	c.ClipboardSync = enabled
	c.mu.Unlock()
	return c.Save()
}

// IsClipboardSyncEnabled returns whether clipboard synchronization is enabled.
func (c *Config) IsClipboardSyncEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ClipboardSync
}

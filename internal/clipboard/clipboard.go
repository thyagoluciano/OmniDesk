package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.design/x/clipboard"
	"omnidesk/internal/config"
	"omnidesk/internal/discovery"
)

// PeerProvider supplies active peers and configuration for broadcasting.
type PeerProvider interface {
	GetOnlineTrustedPeers() []discovery.DiscoveredPeer
}

// Engine coordinates clipboard watching, anti-echo filtering, and remote broadcast.
type Engine struct {
	mu          sync.RWMutex
	cfg         *config.Config
	peers       PeerProvider
	antiEcho    map[string]time.Time // sha256 -> expiration time
	initialized bool
	httpClient  *http.Client
	cancelWatch context.CancelFunc
}

// NewEngine creates a new clipboard management engine.
func NewEngine(cfg *config.Config, peers PeerProvider) *Engine {
	return &Engine{
		cfg:        cfg,
		peers:      peers,
		antiEcho:   make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

// HashText computes SHA-256 for a given text.
func HashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

// Start begins OS clipboard monitoring.
func (e *Engine) Start(ctx context.Context) error {
	if err := clipboard.Init(); err != nil {
		log.Printf("[clipboard] warning: OS clipboard init failed (running headless or without display?): %v", err)
		return nil
	}
	e.initialized = true

	watchCtx, cancel := context.WithCancel(ctx)
	e.cancelWatch = cancel

	ch := clipboard.Watch(watchCtx, clipboard.FmtText)
	go func() {
		for {
			select {
			case <-watchCtx.Done():
				return
			case data, ok := <-ch:
				if !ok {
					return
				}
				text := string(data.Bytes)
				if text != "" {
					e.HandleLocalCopy(text)
				}
			}
		}
	}()

	// Background routine to prune old anti-echo hashes
	go e.pruneLoop(watchCtx)

	log.Printf("[clipboard] monitoring active")
	return nil
}

// Stop terminates clipboard watching.
func (e *Engine) Stop() {
	if e.cancelWatch != nil {
		e.cancelWatch()
	}
}

// IsSyncEnabled returns if clipboard sync is active.
func (e *Engine) IsSyncEnabled() bool {
	return e.cfg.IsClipboardSyncEnabled()
}

// HandleLocalCopy is called whenever text is copied locally on this device.
func (e *Engine) HandleLocalCopy(text string) {
	if !e.IsSyncEnabled() {
		return
	}

	h := HashText(text)

	e.mu.Lock()
	exp, exists := e.antiEcho[h]
	if exists && time.Now().Before(exp) {
		// This text was recently received from a peer; do not echo back!
		e.mu.Unlock()
		return
	}
	// Record this hash locally so we don't re-broadcast duplicates in short succession
	e.antiEcho[h] = time.Now().Add(5 * time.Second)
	e.mu.Unlock()

	log.Printf("[clipboard] texto copiado localmente (%d bytes), transmitindo para dispositivos pareados...", len(text))

	// Broadcast to all trusted peers
	go e.broadcastText(text)
}

// InjectRemoteClipboard receives text from a remote peer and writes it to OS clipboard.
func (e *Engine) InjectRemoteClipboard(text string, senderID string) error {
	if !e.IsSyncEnabled() {
		return nil
	}

	h := HashText(text)

	e.mu.Lock()
	// Mark in anti-echo cache before writing so the local watcher ignores it
	e.antiEcho[h] = time.Now().Add(5 * time.Second)
	e.mu.Unlock()

	if e.initialized {
		writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = clipboard.Write(writeCtx, clipboard.FmtText, []byte(text))
	}

	log.Printf("[clipboard] texto recebido e injetado do dispositivo %s (%d bytes)", senderID, len(text))
	return nil
}

func (e *Engine) broadcastText(text string) {
	trusted := e.cfg.ListTrustedDevices()
	if len(trusted) == 0 {
		return
	}

	payload := map[string]string{
		"sender_id": e.cfg.DeviceID,
		"text":      text,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, dev := range trusted {
		targetAddr := dev.LastAddr
		if targetAddr == "" || strings.HasPrefix(targetAddr, "127.0.0.1") {
			continue
		}

		go func(p config.TrustedDevice, addr string) {
			url := fmt.Sprintf("http://%s/api/v1/clipboard", addr)
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-OmniDesk-Device-ID", e.cfg.DeviceID)
			req.Header.Set("X-OmniDesk-Token", p.Token)

			resp, err := e.httpClient.Do(req)
			if err != nil {
				log.Printf("[clipboard] erro ao enviar para %s (%s): %v", p.Name, addr, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Printf("[clipboard] enviado com sucesso para %s (%s) [%d bytes]", p.Name, addr, len(text))
			} else {
				log.Printf("[clipboard] %s recusou o texto (HTTP %d)", p.Name, resp.StatusCode)
			}
		}(dev, targetAddr)
	}
}

func (e *Engine) pruneLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			now := time.Now()
			for h, exp := range e.antiEcho {
				if now.After(exp) {
					delete(e.antiEcho, h)
				}
			}
			e.mu.Unlock()
		}
	}
}

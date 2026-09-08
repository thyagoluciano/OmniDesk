package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"crossover/internal/config"
	"crossover/internal/discovery"
	"crossover/internal/pairing"
	"crossover/web"
)

// ClipboardHandler handles incoming remote clipboard text.
type ClipboardHandler interface {
	InjectRemoteClipboard(text string, senderID string) error
	IsSyncEnabled() bool
}

// FileNotificationHandler triggers native desktop notifications for received files.
type FileNotificationHandler interface {
	NotifyFileReceived(fileName, senderName string, sizeBytes int64)
}

// Server handles all inbound HTTP requests for Crossover.
type Server struct {
	mu          sync.RWMutex
	cfg         *config.Config
	pairingMgr  *pairing.Manager
	clipHandler ClipboardHandler
	notify      FileNotificationHandler
	httpServer  *http.Server
	listenAddr  string
	node        *Node
}

// NewServer initializes the HTTP API server.
func NewServer(cfg *config.Config, pMgr *pairing.Manager, cHandler ClipboardHandler, notify FileNotificationHandler) *Server {
	s := &Server{
		cfg:         cfg,
		pairingMgr:  pMgr,
		clipHandler: cHandler,
		notify:      notify,
	}
	return s
}

// Start runs the HTTP listener on the configured port.
func (s *Server) Start(port int) error {
	mux := http.NewServeMux()

	// Web UI static files
	uiFS := http.FileServer(web.GetFileSystem())
	mux.Handle("/ui/", http.StripPrefix("/ui/", uiFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/ui/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Public endpoints (for pairing handshake)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/pair/request", s.handlePairRequest)
	mux.HandleFunc("/api/v1/pair/confirm", s.handlePairConfirm)
	mux.HandleFunc("/api/v1/pair/approve-pin", s.handleApprovePIN)
	mux.HandleFunc("/api/v1/pair/pending", s.handlePendingSessions)

	// Web UI management endpoints
	mux.HandleFunc("/api/v1/devices", s.handleDevices)
	mux.HandleFunc("/api/v1/devices/send", s.handleDevicesSend)
	mux.HandleFunc("/api/v1/devices/pair", s.handleDevicesPair)
	mux.HandleFunc("/api/v1/clipboard/toggle", s.handleClipboardToggle)

	// Protected endpoints (require trusted device token)
	mux.HandleFunc("/api/v1/clipboard", s.authMiddleware(s.handleClipboard))
	mux.HandleFunc("/api/v1/files/upload", s.authMiddleware(s.handleFileUpload))

	s.listenAddr = fmt.Sprintf("0.0.0.0:%d", port)
	s.httpServer = &http.Server{
		Addr:         s.listenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Minute, // Allow large file streaming
		WriteTimeout: 30 * time.Minute,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[server] listen error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devID := r.Header.Get("X-Crossover-Device-ID")
		token := r.Header.Get("X-Crossover-Token")

		if devID == "" || token == "" {
			http.Error(w, "missing authentication headers", http.StatusUnauthorized)
			return
		}

		if !s.cfg.IsTrusted(devID, token) {
			http.Error(w, "unauthorized: device not paired or invalid token", http.StatusForbidden)
			return
		}

		// Dynamically update peer address and last seen based on the incoming TCP connection!
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" && host != "127.0.0.1" && host != "::1" {
			if dev, ok := s.cfg.GetTrustedDevice(devID); ok {
				dev.LastSeen = time.Now()
				dev.LastAddr = fmt.Sprintf("%s:%d", host, s.cfg.ListenPort)
				_ = s.cfg.AddTrustedDevice(dev)
			}
		}

		next(w, r)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"device_id":      s.cfg.DeviceID,
		"device_name":    s.cfg.DeviceName,
		"clipboard_sync": s.cfg.IsClipboardSyncEnabled(),
		"port":           s.cfg.ListenPort,
		"download_dir":   s.cfg.DownloadDir,
		"status":         "online",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePairRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req pairing.PairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Always resolve the real remote IP from the HTTP connection if req.RequesterAddr is localhost
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" && host != "127.0.0.1" && host != "::1" {
		req.RequesterAddr = fmt.Sprintf("%s:%d", host, s.cfg.ListenPort)
	}

	resp, err := s.pairingMgr.HandlePairRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePairConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req pairing.ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp, err := s.pairingMgr.HandleConfirmRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type ApprovePINRequest struct {
	PIN string `json:"pin"`
}

func (s *Server) handleApprovePIN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApprovePINRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sess, err := s.pairingMgr.ApproveByPIN(req.PIN)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"device_name": sess.RequesterName,
		"device_id":   sess.RequesterID,
	})
}

func (s *Server) handlePendingSessions(w http.ResponseWriter, r *http.Request) {
	pending := s.pairingMgr.GetPendingSessions()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pending)
}

type ClipboardPayload struct {
	SenderID string `json:"sender_id"`
	Text     string `json:"text"`
}

func (s *Server) handleClipboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.clipHandler == nil || !s.clipHandler.IsSyncEnabled() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"sync_disabled"}`))
		return
	}

	var payload ClipboardPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := s.clipHandler.InjectRemoteClipboard(payload.Text, payload.SenderID); err != nil {
		log.Printf("[server] failed to inject clipboard: %v", err)
		http.Error(w, "clipboard injection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	senderID := r.Header.Get("X-Crossover-Device-ID")
	senderDev, _ := s.cfg.GetTrustedDevice(senderID)
	senderName := senderDev.Name
	if senderName == "" {
		senderName = "Dispositivo Crossover"
	}

	fileName := r.URL.Query().Get("filename")
	if fileName == "" {
		fileName = fmt.Sprintf("file-%d.bin", time.Now().Unix())
	}
	fileName = filepath.Base(fileName) // Sanitize filename

	downloadDir := s.cfg.DownloadDir
	_ = os.MkdirAll(downloadDir, 0755)

	// Deduplicate filename if already exists
	targetPath := resolveUniqueFilename(downloadDir, fileName)

	out, err := os.Create(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create file: %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Stream directly to disk without loading entirely in memory
	written, err := io.Copy(out, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("file write error: %v", err), http.StatusInternalServerError)
		return
	}

	if s.notify != nil {
		s.notify.NotifyFileReceived(filepath.Base(targetPath), senderName, written)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "received",
		"filename":    filepath.Base(targetPath),
		"bytes_saved": written,
	})
}

// resolveUniqueFilename handles name collisions like "photo.png" -> "photo (1).png"
func resolveUniqueFilename(dir, filename string) string {
	dest := filepath.Join(dir, filename)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}

	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	counter := 1

	for {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, counter, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		counter++
	}
}

type DeviceItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LastAddr string `json:"last_addr"`
	LastSeen string `json:"last_seen"`
	IsOnline bool   `json:"is_online"`
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	trustedList := s.cfg.ListTrustedDevices()
	onlineMap := make(map[string]bool)
	var discoveredPeers []discovery.DiscoveredPeer

	if s.node != nil {
		for _, p := range s.node.GetOnlineTrustedPeers() {
			onlineMap[p.ID] = true
		}
		discoveredPeers = s.node.GetAllDiscoveredPeers()
	}

	trustedItems := make([]DeviceItem, 0, len(trustedList))
	for _, dev := range trustedList {
		lastSeen := "Nunca"
		if !dev.LastSeen.IsZero() {
			lastSeen = dev.LastSeen.Format("02/01 15:04:05")
		}
		trustedItems = append(trustedItems, DeviceItem{
			ID:       dev.ID,
			Name:     dev.Name,
			LastAddr: dev.LastAddr,
			LastSeen: lastSeen,
			IsOnline: onlineMap[dev.ID],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"trusted":    trustedItems,
		"discovered": discoveredPeers,
	})
}

func (s *Server) handleClipboardToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	_ = s.cfg.SetClipboardSync(body.Enabled)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"clipboard_sync": body.Enabled,
	})
}

func (s *Server) handleDevicesPair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Target == "" {
		http.Error(w, "target is required", http.StatusBadRequest)
		return
	}

	target := body.Target
	var targetAddr string
	if strings.Contains(target, ":") {
		targetAddr = target
	} else if net.ParseIP(target) != nil {
		targetAddr = fmt.Sprintf("%s:%d", target, s.cfg.ListenPort)
	} else if s.node != nil {
		for _, p := range s.node.GetAllDiscoveredPeers() {
			if strings.EqualFold(p.Name, target) || strings.EqualFold(p.ID, target) {
				targetAddr = p.Addr
				break
			}
		}
	}

	if targetAddr == "" {
		http.Error(w, fmt.Sprintf("dispositivo '%s' não encontrado na rede", target), http.StatusNotFound)
		return
	}

	localIP := getOutboundIPForServer(targetAddr)
	myAddr := fmt.Sprintf("%s:%d", localIP, s.cfg.ListenPort)

	sess, err := s.pairingMgr.InitiatePairing(targetAddr, myAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// In background, complete pairing once approved
	go func() {
		for i := 0; i < 45; i++ {
			time.Sleep(2 * time.Second)
			if err := s.pairingMgr.CompletePairing(targetAddr, sess); err == nil {
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pin":    sess.PIN,
		"target": targetAddr,
	})
}

func (s *Server) handleDevicesSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetID := r.URL.Query().Get("target")
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = fmt.Sprintf("upload-%d.bin", time.Now().Unix())
	}
	filename = filepath.Base(filename)

	targetDev, ok := s.cfg.GetTrustedDevice(targetID)
	if !ok {
		http.Error(w, "target device not trusted or found", http.StatusBadRequest)
		return
	}

	targetAddr := targetDev.LastAddr
	if s.node != nil {
		for _, p := range s.node.GetOnlineTrustedPeers() {
			if p.ID == targetDev.ID {
				targetAddr = p.Addr
				break
			}
		}
	}

	if targetAddr == "" {
		http.Error(w, "target device is offline or has no known address", http.StatusBadGateway)
		return
	}

	if s.node == nil || s.node.TransferCli == nil {
		http.Error(w, "transfer client not initialized", http.StatusInternalServerError)
		return
	}

	res, err := s.node.TransferCli.SendStream(r.Context(), r.Body, r.ContentLength, filename, targetAddr, targetDev.ID, targetDev.Token, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func getOutboundIPForServer(target string) string {
	conn, err := net.Dial("udp", target)
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}


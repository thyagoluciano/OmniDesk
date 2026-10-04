package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"omnidesk/internal/clipboard"
	"omnidesk/internal/config"
	"omnidesk/internal/pairing"
)

type mockClipHandler struct {
	injected []string
}

func (m *mockClipHandler) InjectRemoteClipboard(text string, senderID string) error {
	m.injected = append(m.injected, text)
	return nil
}

func (m *mockClipHandler) IsSyncEnabled() bool {
	return true
}

func (m *mockClipHandler) GetHistory() []clipboard.HistoryEntry {
	return nil
}

func (m *mockClipHandler) RemoveHistoryEntry(id string) {}

func TestServerPairingAndAuth(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		DeviceID:          "node-b",
		DeviceName:        "Linux-Desktop",
		ListenPort:        24850,
		DownloadDir:       tempDir,
		ClipboardSync:     true,
		FilesSync:         true,
		InputShareEnabled: true,
		TrustedDevices:    make(map[string]config.TrustedDevice),
	}

	pMgr := pairing.NewManager(cfg)
	clip := &mockClipHandler{}
	srv := NewServer(cfg, pMgr, clip, nil)

	// Test Status endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	srv.handleStatus(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Test Inbound Pairing Request
	pairReq := pairing.PairRequest{
		RequesterID:   "node-a",
		RequesterName: "MacBook-Pro",
		RequesterAddr: "192.168.1.50:24850",
	}
	body, _ := json.Marshal(pairReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pair/request", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.handlePairRequest(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from pair request, got %d", w.Code)
	}

	var pairResp pairing.PairResponse
	_ = json.Unmarshal(w.Body.Bytes(), &pairResp)
	if pairResp.SessionID == "" || len(pairResp.PIN) != 6 {
		t.Fatalf("invalid pair response: %+v", pairResp)
	}

	// Responder approves session
	if err := pMgr.ApproveSession(pairResp.SessionID); err != nil {
		t.Fatalf("failed to approve session: %v", err)
	}

	// Requester confirms PIN
	confReq := pairing.ConfirmRequest{
		SessionID:   pairResp.SessionID,
		PIN:         pairResp.PIN,
		RequesterID: "node-a",
	}
	body, _ = json.Marshal(confReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pair/confirm", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.handlePairConfirm(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from pair confirm, got %d", w.Code)
	}

	var confResp pairing.ConfirmResponse
	_ = json.Unmarshal(w.Body.Bytes(), &confResp)
	if !confResp.Success || confResp.AuthToken == "" {
		t.Fatalf("confirmation failed: %+v", confResp)
	}

	// Now node-a should be trusted
	if !cfg.IsTrusted("node-a", confResp.AuthToken) {
		t.Fatalf("expected node-a to be trusted in config")
	}

	// Test protected clipboard endpoint with valid token
	clipPayload := ClipboardPayload{
		SenderID: "node-a",
		Text:     "Olá mundo!",
	}
	body, _ = json.Marshal(clipPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/clipboard", bytes.NewReader(body))
	req.Header.Set("X-OmniDesk-Device-ID", "node-a")
	req.Header.Set("X-OmniDesk-Token", confResp.AuthToken)
	w = httptest.NewRecorder()
	srv.authMiddleware(srv.handleClipboard)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from authenticated clipboard, got %d", w.Code)
	}
	if len(clip.injected) != 1 || clip.injected[0] != "Olá mundo!" {
		t.Errorf("unexpected clipboard text: %v", clip.injected)
	}

	// Test protected endpoint with invalid token
	req = httptest.NewRequest(http.MethodPost, "/api/v1/clipboard", bytes.NewReader(body))
	req.Header.Set("X-OmniDesk-Device-ID", "node-a")
	req.Header.Set("X-OmniDesk-Token", "fake-token")
	w = httptest.NewRecorder()
	srv.authMiddleware(srv.handleClipboard)(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for bad token, got %d", w.Code)
	}
}

func TestResolveUniqueFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Initial file
	path1 := resolveUniqueFilename(tempDir, "doc.pdf")
	if filepath.Base(path1) != "doc.pdf" {
		t.Errorf("expected doc.pdf, got %s", path1)
	}
	_ = os.WriteFile(path1, []byte("test"), 0644)

	// Second file with same name
	path2 := resolveUniqueFilename(tempDir, "doc.pdf")
	if filepath.Base(path2) != "doc (1).pdf" {
		t.Errorf("expected doc (1).pdf, got %s", path2)
	}
	_ = os.WriteFile(path2, []byte("test2"), 0644)

	// Third file with same name
	path3 := resolveUniqueFilename(tempDir, "doc.pdf")
	if filepath.Base(path3) != "doc (2).pdf" {
		t.Errorf("expected doc (2).pdf, got %s", path3)
	}
}

func TestServerQRPairing(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		DeviceID:       "desktop-1",
		DeviceName:     "MacBook Pro",
		ListenPort:     24850,
		DownloadDir:    tempDir,
		ClipboardSync:  true,
		TrustedDevices: make(map[string]config.TrustedDevice),
	}

	pMgr := pairing.NewManager(cfg)
	srv := NewServer(cfg, pMgr, &mockClipHandler{}, nil)

	// 1. Generate QR Code payload
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pair/qr/generate", nil)
	w := httptest.NewRecorder()
	srv.handlePairQRGenerate(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from pair/qr/generate, got %d", w.Code)
	}

	var qrResp struct {
		V     int      `json:"v"`
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Addrs []string `json:"addrs"`
		PIN   string   `json:"pin"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &qrResp); err != nil {
		t.Fatalf("invalid QR response: %v", err)
	}
	if qrResp.ID != "desktop-1" || len(qrResp.PIN) != 6 {
		t.Fatalf("invalid QR payload fields: %+v", qrResp)
	}

	// 2. Check QR Status before redeem -> not approved
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pair/qr/status?pin="+qrResp.PIN, nil)
	w = httptest.NewRecorder()
	srv.handlePairQRStatus(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from pair/qr/status, got %d", w.Code)
	}
	var statusResp struct {
		Approved bool `json:"approved"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &statusResp)
	if statusResp.Approved {
		t.Fatal("expected approved to be false before redeem")
	}

	// 3. Mobile redeems QR code
	redeemPayload := PairQRRedeemRequest{
		PIN:           qrResp.PIN,
		RequesterID:   "mobile-iphone",
		RequesterName: "iPhone 15 Pro",
		RequesterAddr: "192.168.1.100:24851",
	}
	body, _ := json.Marshal(redeemPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pair/qr/redeem", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.handlePairQRRedeem(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from pair/qr/redeem, got %d: %s", w.Code, w.Body.String())
	}

	var redeemResp struct {
		Success   bool   `json:"success"`
		AuthToken string `json:"auth_token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &redeemResp)
	if !redeemResp.Success || redeemResp.AuthToken == "" {
		t.Fatalf("expected success and auth_token from redeem, got: %+v", redeemResp)
	}

	// 4. Check QR Status after redeem -> approved!
	req = httptest.NewRequest(http.MethodGet, "/api/v1/pair/qr/status?pin="+qrResp.PIN, nil)
	w = httptest.NewRecorder()
	srv.handlePairQRStatus(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &statusResp)
	if !statusResp.Approved {
		t.Fatal("expected approved to be true after redeem")
	}

	// 5. Verify mobile device is trusted in config
	if !cfg.IsTrusted("mobile-iphone", redeemResp.AuthToken) {
		t.Fatal("expected mobile-iphone to be trusted in config")
	}
}

func TestNormalizeRequesterAddr(t *testing.T) {
	cases := []struct{ reported, remote, want string }{
		{"0.0.0.0:24851", "192.168.100.50:51000", "192.168.100.50:24851"},
		{"", "192.168.100.50:51000", "192.168.100.50:24851"},
		{"127.0.0.1:24851", "192.168.100.50:51000", "192.168.100.50:24851"},
		{"192.168.100.60:24851", "192.168.100.50:51000", "192.168.100.60:24851"},
	}
	for _, c := range cases {
		if got := normalizeRequesterAddr(c.reported, c.remote); got != c.want {
			t.Errorf("normalizeRequesterAddr(%q, %q) = %q, want %q", c.reported, c.remote, got, c.want)
		}
	}
}

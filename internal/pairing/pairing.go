package pairing

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"omnidesk/internal/config"
)

// Session represents an active pairing handshake.
type Session struct {
	ID            string    `json:"session_id"`
	PIN           string    `json:"pin"`
	RequesterID   string    `json:"requester_id"`
	RequesterName string    `json:"requester_name"`
	RequesterAddr string    `json:"requester_addr"`
	ResponderID   string    `json:"responder_id"`
	ResponderName string    `json:"responder_name"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Approved      bool      `json:"approved"`
	AuthToken     string    `json:"auth_token,omitempty"`
}

type PairRequest struct {
	RequesterID   string `json:"requester_id"`
	RequesterName string `json:"requester_name"`
	RequesterAddr string `json:"requester_addr"`
}

type PairResponse struct {
	SessionID     string `json:"session_id"`
	PIN           string `json:"pin"`
	ResponderID   string `json:"responder_id"`
	ResponderName string `json:"responder_name"`
}

type ConfirmRequest struct {
	SessionID   string `json:"session_id"`
	PIN         string `json:"pin"`
	RequesterID string `json:"requester_id"`
}

type ConfirmResponse struct {
	Success   bool   `json:"success"`
	AuthToken string `json:"auth_token"`
	Message   string `json:"message,omitempty"`
}

// Manager coordinates pairing handshakes between nodes.
type Manager struct {
	mu         sync.RWMutex
	cfg        *config.Config
	sessions   map[string]*Session
	httpClient *http.Client
	OnPrompt   func(session *Session) // Callback when a pairing request needs user attention
}

// NewManager creates a new pairing manager.
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*Session),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GeneratePIN creates a cryptographically random 6-digit PIN.
func GeneratePIN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func generateToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// InitiatePairing starts the pairing process toward a remote peer address.
func (m *Manager) InitiatePairing(peerAddr, myAddr string) (*Session, error) {
	reqBody := PairRequest{
		RequesterID:   m.cfg.DeviceID,
		RequesterName: m.cfg.DeviceName,
		RequesterAddr: myAddr,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s/api/v1/pair/request", peerAddr)
	resp, err := m.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to contact peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer rejected pair request with status: %d", resp.StatusCode)
	}

	var pairResp PairResponse
	if err := json.NewDecoder(resp.Body).Decode(&pairResp); err != nil {
		return nil, fmt.Errorf("invalid pair response: %w", err)
	}

	session := &Session{
		ID:            pairResp.SessionID,
		PIN:           pairResp.PIN,
		RequesterID:   m.cfg.DeviceID,
		RequesterName: m.cfg.DeviceName,
		RequesterAddr: myAddr,
		ResponderID:   pairResp.ResponderID,
		ResponderName: pairResp.ResponderName,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(2 * time.Minute),
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	return session, nil
}

// HandlePairRequest handles an inbound request from a peer wanting to pair.
func (m *Manager) HandlePairRequest(req PairRequest) (*PairResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean up expired sessions
	now := time.Now()
	for id, s := range m.sessions {
		if now.After(s.ExpiresAt) {
			delete(m.sessions, id)
		}
	}

	pin := GeneratePIN()
	sessID := generateToken()

	session := &Session{
		ID:            sessID,
		PIN:           pin,
		RequesterID:   req.RequesterID,
		RequesterName: req.RequesterName,
		RequesterAddr: req.RequesterAddr,
		ResponderID:   m.cfg.DeviceID,
		ResponderName: m.cfg.DeviceName,
		CreatedAt:     now,
		ExpiresAt:     now.Add(2 * time.Minute),
	}
	m.sessions[sessID] = session

	if m.OnPrompt != nil {
		go m.OnPrompt(session)
	}

	return &PairResponse{
		SessionID:     sessID,
		PIN:           pin,
		ResponderID:   m.cfg.DeviceID,
		ResponderName: m.cfg.DeviceName,
	}, nil
}

// ApproveSession marks the session approved on the responder device.
func (m *Manager) ApproveSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return errors.New("session not found or expired")
	}
	if time.Now().After(sess.ExpiresAt) {
		delete(m.sessions, sessionID)
		return errors.New("session expired")
	}

	sess.Approved = true
	sess.AuthToken = generateToken()

	// Save requester as trusted
	_ = m.cfg.AddTrustedDevice(config.TrustedDevice{
		ID:       sess.RequesterID,
		Name:     sess.RequesterName,
		Token:    sess.AuthToken,
		AddedAt:  time.Now(),
		LastSeen: time.Now(),
		LastAddr: sess.RequesterAddr,
	})

	return nil
}

// ApproveByPIN approves the active session matching the given 6-digit PIN.
func (m *Manager) ApproveByPIN(pin string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanPIN := strings.ReplaceAll(strings.TrimSpace(pin), " ", "")

	for _, sess := range m.sessions {
		if sess.PIN == cleanPIN && time.Now().Before(sess.ExpiresAt) {
			sess.Approved = true
			sess.AuthToken = generateToken()

			_ = m.cfg.AddTrustedDevice(config.TrustedDevice{
				ID:       sess.RequesterID,
				Name:     sess.RequesterName,
				Token:    sess.AuthToken,
				AddedAt:  time.Now(),
				LastSeen: time.Now(),
				LastAddr: sess.RequesterAddr,
			})
			return sess, nil
		}
	}
	return nil, errors.New("nenhuma sessão pendente encontrada com este PIN ou PIN expirado")
}

// HandleConfirmRequest responds to the confirmation endpoint.
func (m *Manager) HandleConfirmRequest(req ConfirmRequest) (*ConfirmResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[req.SessionID]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return &ConfirmResponse{Success: false, Message: "session expired or invalid"}, nil
	}

	if sess.PIN != req.PIN {
		return &ConfirmResponse{Success: false, Message: "invalid PIN"}, nil
	}

	if !sess.Approved {
		return &ConfirmResponse{Success: false, Message: "pending user approval on responder device"}, nil
	}

	token := sess.AuthToken
	delete(m.sessions, req.SessionID)

	return &ConfirmResponse{
		Success:   true,
		AuthToken: token,
	}, nil
}

// CompletePairing is called by the requester after responder approved.
func (m *Manager) CompletePairing(peerAddr string, session *Session) error {
	reqBody := ConfirmRequest{
		SessionID:   session.ID,
		PIN:         session.PIN,
		RequesterID: m.cfg.DeviceID,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/api/v1/pair/confirm", peerAddr)
	resp, err := m.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to complete pairing: %w", err)
	}
	defer resp.Body.Close()

	var confResp ConfirmResponse
	if err := json.NewDecoder(resp.Body).Decode(&confResp); err != nil {
		return err
	}

	if !confResp.Success {
		return fmt.Errorf("pairing rejected: %s", confResp.Message)
	}

	// Persist responder as trusted
	return m.cfg.AddTrustedDevice(config.TrustedDevice{
		ID:       session.ResponderID,
		Name:     session.ResponderName,
		Token:    confResp.AuthToken,
		AddedAt:  time.Now(),
		LastSeen: time.Now(),
		LastAddr: peerAddr,
	})
}

// GetPendingSessions returns active unexpired pairing sessions.
func (m *Manager) GetPendingSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Session
	now := time.Now()
	for _, s := range m.sessions {
		if now.Before(s.ExpiresAt) {
			list = append(list, s)
		}
	}
	return list
}

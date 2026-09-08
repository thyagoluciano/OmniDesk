package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const ServiceName = "_crossover._tcp"
const Domain = "local."

// DiscoveredPeer represents an active peer announced via mDNS.
type DiscoveredPeer struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Addr     string    `json:"addr"` // e.g., 192.168.1.50:24850
	IPs      []net.IP  `json:"ips"`
	Port     int       `json:"port"`
	LastSeen time.Time `json:"last_seen"`
}

// PeerHandler receives notifications when peers appear or update.
type PeerHandler interface {
	OnPeerFound(peer DiscoveredPeer)
	OnPeerLost(peerID string)
}

// Service manages announcing this node and browsing for other crossover nodes.
type Service struct {
	mu         sync.RWMutex
	deviceID   string
	deviceName string
	port       int
	peers      map[string]DiscoveredPeer
	handler    PeerHandler
	server     *zeroconf.Server
	cancelFunc context.CancelFunc
}

// NewService creates a new discovery service instance.
func NewService(deviceID, deviceName string, port int, handler PeerHandler) *Service {
	return &Service{
		deviceID:   deviceID,
		deviceName: deviceName,
		port:       port,
		peers:      make(map[string]DiscoveredPeer),
		handler:    handler,
	}
}

// Start registers this node via mDNS and starts browsing for peers in the background.
func (s *Service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	txtRecords := []string{
		fmt.Sprintf("id=%s", s.deviceID),
		fmt.Sprintf("name=%s", s.deviceName),
		"v=1",
	}

	instanceName := fmt.Sprintf("%s-%s", s.deviceName, s.deviceID)
	server, err := zeroconf.Register(instanceName, ServiceName, Domain, s.port, txtRecords, nil)
	if err != nil {
		return fmt.Errorf("failed to register zeroconf service: %w", err)
	}
	s.server = server

	go s.browseLoop(ctx)
	return nil
}

// Stop unregisters mDNS server and stops browsing.
func (s *Service) Stop() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	if s.server != nil {
		s.server.Shutdown()
	}
}

func (s *Service) browseLoop(ctx context.Context) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Printf("[discovery] failed to initialize resolver: %v", err)
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	runBrowse := func() {
		entries := make(chan *zeroconf.ServiceEntry)
		browseCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		go func() {
			for entry := range entries {
				s.handleEntry(entry)
			}
		}()

		if err := resolver.Browse(browseCtx, ServiceName, Domain, entries); err != nil {
			log.Printf("[discovery] browse error: %v", err)
		}
	}

	runBrowse()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runBrowse()
			s.pruneInactivePeers()
		}
	}
}

func (s *Service) handleEntry(entry *zeroconf.ServiceEntry) {
	var id, name string
	for _, text := range entry.Text {
		parts := strings.SplitN(text, "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "id":
				id = parts[1]
			case "name":
				name = parts[1]
			}
		}
	}

	// Ignore self
	if id == "" || id == s.deviceID {
		return
	}

	var primaryIP net.IP
	if len(entry.AddrIPv4) > 0 {
		primaryIP = entry.AddrIPv4[0]
	} else if len(entry.AddrIPv6) > 0 {
		primaryIP = entry.AddrIPv6[0]
	} else {
		return
	}

	peer := DiscoveredPeer{
		ID:       id,
		Name:     name,
		Addr:     fmt.Sprintf("%s:%d", primaryIP.String(), entry.Port),
		IPs:      entry.AddrIPv4,
		Port:     entry.Port,
		LastSeen: time.Now(),
	}

	s.mu.Lock()
	s.peers[id] = peer
	s.mu.Unlock()

	if s.handler != nil {
		s.handler.OnPeerFound(peer)
	}
}

func (s *Service) pruneInactivePeers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := time.Now().Add(-90 * time.Second)
	for id, peer := range s.peers {
		if peer.LastSeen.Before(threshold) {
			delete(s.peers, id)
			if s.handler != nil {
				s.handler.OnPeerLost(id)
			}
		}
	}
}

// GetPeers returns a snapshot of discovered peers.
func (s *Service) GetPeers() []DiscoveredPeer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]DiscoveredPeer, 0, len(s.peers))
	for _, peer := range s.peers {
		list = append(list, peer)
	}
	return list
}

package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"omnidesk/internal/clipboard"
	"omnidesk/internal/config"
	"omnidesk/internal/discovery"
	"omnidesk/internal/inputshare"
	"omnidesk/internal/notify"
	"omnidesk/internal/pairing"
	"omnidesk/internal/transfer"
)

// Node represents a running OmniDesk instance with all local services.
type Node struct {
	mu          sync.RWMutex
	Cfg         *config.Config
	Discovery   *discovery.Service
	PairingMgr  *pairing.Manager
	ClipEngine  *clipboard.Engine
	TransferCli *transfer.Client
	Notifier    *notify.Notifier
	InputMgr    *inputshare.Manager
	Server      *Server
	onlinePeers map[string]discovery.DiscoveredPeer
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewNode initializes all services for a node.
func NewNode(cfg *config.Config) *Node {
	n := &Node{
		Cfg:         cfg,
		onlinePeers: make(map[string]discovery.DiscoveredPeer),
		Notifier:    notify.NewNotifier(),
		TransferCli: transfer.NewClient(cfg.DeviceID),
	}

	n.PairingMgr = pairing.NewManager(cfg)
	n.PairingMgr.OnPrompt = func(sess *pairing.Session) {
		msg := fmt.Sprintf("Pareamento solicitado por %s. PIN: %s", sess.RequesterName, sess.PIN)
		_ = n.Notifier.SendNotification("OmniDesk - Pareamento", msg)
	}

	n.ClipEngine = clipboard.NewEngine(cfg, n)
	n.InputMgr = inputshare.NewManager(cfg, n)
	n.Server = NewServer(cfg, n.PairingMgr, n.ClipEngine, n.Notifier)
	n.Server.node = n
	n.Discovery = discovery.NewService(cfg.DeviceID, cfg.DeviceName, cfg.ListenPort, n)

	return n
}

// Start launches the node HTTP server, mDNS discovery, and clipboard monitor.
func (n *Node) Start(ctx context.Context) error {
	n.ctx, n.cancel = context.WithCancel(ctx)

	// Start local HTTP server
	if err := n.Server.Start(n.Cfg.ListenPort); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Start mDNS discovery and announcement
	if err := n.Discovery.Start(n.ctx); err != nil {
		log.Printf("[node] mDNS warning: %v", err)
	}

	// Start OS clipboard watching
	if err := n.ClipEngine.Start(n.ctx); err != nil {
		log.Printf("[node] clipboard watcher warning: %v", err)
	}

	// Start input-sharing (KVM) capture/injection backend, if the platform
	// supports one. Degrades gracefully otherwise, exactly like clipboard.
	if err := n.InputMgr.Start(n.ctx); err != nil {
		log.Printf("[node] inputshare warning: %v", err)
	}

	// Start Subnet Probe loop to discover peers over unicast HTTP (bypasses Wi-Fi mDNS blocks)
	go n.subnetProbeLoop(n.ctx)

	log.Printf("[node] OmniDesk running as '%s' (ID: %s) on port %d", n.Cfg.DeviceName, n.Cfg.DeviceID, n.Cfg.ListenPort)
	return nil
}

func (n *Node) subnetProbeLoop(ctx context.Context) {
	sweep := func() {
		probeCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		peers := discovery.ProbeSubnet(probeCtx, n.Cfg.ListenPort, n.Cfg.DeviceID)
		for _, p := range peers {
			n.OnPeerFound(p)
		}
	}

	// Initial sweep shortly after start
	time.Sleep(1 * time.Second)
	sweep()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}

// Stop terminates all node background services.
func (n *Node) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
	if n.Discovery != nil {
		n.Discovery.Stop()
	}
	if n.ClipEngine != nil {
		n.ClipEngine.Stop()
	}
	if n.InputMgr != nil {
		n.InputMgr.Stop()
	}
	if n.Server != nil {
		_ = n.Server.Stop()
	}
}

// OnPeerFound is called by the discovery service when a peer is discovered.
func (n *Node) OnPeerFound(peer discovery.DiscoveredPeer) {
	n.mu.Lock()
	n.onlinePeers[peer.ID] = peer
	n.mu.Unlock()

	// Update last seen & last addr in config if trusted
	if dev, ok := n.Cfg.GetTrustedDevice(peer.ID); ok {
		dev.LastSeen = time.Now()
		dev.LastAddr = peer.Addr
		_ = n.Cfg.AddTrustedDevice(dev)

		if peer.ScreenWidth > 0 && peer.ScreenHeight > 0 && n.InputMgr != nil {
			n.InputMgr.UpdatePeerResolution(peer.ID, peer.ScreenWidth, peer.ScreenHeight)
		}
	}
}

// OnPeerLost is called by the discovery service when a peer goes offline.
func (n *Node) OnPeerLost(peerID string) {
	n.mu.Lock()
	delete(n.onlinePeers, peerID)
	n.mu.Unlock()
}

// GetOnlineTrustedPeers returns only online peers that are also in the trusted list.
func (n *Node) GetOnlineTrustedPeers() []discovery.DiscoveredPeer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var result []discovery.DiscoveredPeer
	for _, peer := range n.onlinePeers {
		if _, ok := n.Cfg.GetTrustedDevice(peer.ID); ok {
			result = append(result, peer)
		}
	}
	return result
}

// ResolveAddr implements inputshare.PeerResolver, letting the input-sharing
// manager reuse the same peer discovery/last-known-address data as
// clipboard broadcast and file transfer instead of tracking its own.
func (n *Node) ResolveAddr(peerID string) (string, bool) {
	n.mu.RLock()
	peer, ok := n.onlinePeers[peerID]
	n.mu.RUnlock()
	if ok {
		return peer.Addr, true
	}

	if dev, ok := n.Cfg.GetTrustedDevice(peerID); ok && dev.LastAddr != "" {
		return dev.LastAddr, true
	}
	return "", false
}

// GetAllDiscoveredPeers returns all discovered peers on LAN (paired or unpaired).
func (n *Node) GetAllDiscoveredPeers() []discovery.DiscoveredPeer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var result []discovery.DiscoveredPeer
	for _, peer := range n.onlinePeers {
		result = append(result, peer)
	}
	return result
}

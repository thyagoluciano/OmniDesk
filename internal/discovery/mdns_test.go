package discovery

import (
	"net"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

type mockHandler struct {
	found []DiscoveredPeer
	lost  []string
}

func (m *mockHandler) OnPeerFound(peer DiscoveredPeer) {
	m.found = append(m.found, peer)
}

func (m *mockHandler) OnPeerLost(id string) {
	m.lost = append(m.lost, id)
}

func TestDiscoveryEntryHandling(t *testing.T) {
	handler := &mockHandler{}
	svc := NewService("local-node", "LocalMachine", 24850, handler)

	entry := &zeroconf.ServiceEntry{
		ServiceRecord: zeroconf.ServiceRecord{
			Instance: "RemoteMachine-remote-123",
		},
		Text:     []string{"id=remote-123", "name=RemoteMachine", "v=1"},
		Port:     24850,
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.150")},
	}

	svc.handleEntry(entry)

	peers := svc.GetPeers()
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
	if peers[0].ID != "remote-123" || peers[0].Name != "RemoteMachine" {
		t.Errorf("unexpected peer data: %+v", peers[0])
	}
	if peers[0].Addr != "192.168.1.150:24850" {
		t.Errorf("unexpected peer addr: %s", peers[0].Addr)
	}

	// Verify self is ignored
	selfEntry := &zeroconf.ServiceEntry{
		ServiceRecord: zeroconf.ServiceRecord{
			Instance: "LocalMachine-local-node",
		},
		Text:     []string{"id=local-node", "name=LocalMachine", "v=1"},
		Port:     24850,
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.10")},
	}
	svc.handleEntry(selfEntry)
	if len(svc.GetPeers()) != 1 {
		t.Errorf("expected self to be ignored, peer count: %d", len(svc.GetPeers()))
	}

	// Test prune inactive
	svc.peers["remote-123"] = DiscoveredPeer{
		ID:       "remote-123",
		LastSeen: time.Now().Add(-100 * time.Second),
	}
	svc.pruneInactivePeers()
	if len(svc.GetPeers()) != 0 {
		t.Errorf("expected peer to be pruned")
	}
}

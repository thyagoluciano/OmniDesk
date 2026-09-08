package discovery

import (
	"context"
	"testing"
	"time"
)

func TestSubnetProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Probing with a dummy ID (e.g. "dummy-test-id")
	peers := ProbeSubnet(ctx, 24850, "dummy-test-id")
	t.Logf("Discovered %d peers via subnet probe: %+v", len(peers), peers)
}

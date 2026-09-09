package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type statusResponse struct {
	DeviceID     string `json:"device_id"`
	DeviceName   string `json:"device_name"`
	Status       string `json:"status"`
	ScreenWidth  int    `json:"screen_width"`
	ScreenHeight int    `json:"screen_height"`
}

// ProbeSubnet scans local subnets on the OmniDesk port for active nodes.
// This guarantees discovery even when Wi-Fi routers block mDNS multicast.
func ProbeSubnet(ctx context.Context, port int, myID string) []DiscoveredPeer {
	var peers []DiscoveredPeer
	var mu sync.Mutex

	subnets := getLocalSubnetPrefixes()
	if len(subnets) == 0 {
		return peers
	}

	client := &http.Client{
		Timeout: 400 * time.Millisecond,
	}

	var wg sync.WaitGroup

	for _, prefix := range subnets {
		for i := 1; i <= 254; i++ {
			select {
			case <-ctx.Done():
				break
			default:
			}

			ip := fmt.Sprintf("%s.%d", prefix, i)
			wg.Add(1)

			go func(targetIP string) {
				defer wg.Done()

				targetAddr := fmt.Sprintf("%s:%d", targetIP, port)
				reqURL := fmt.Sprintf("http://%s/api/v1/status", targetAddr)

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
				if err != nil {
					return
				}

				resp, err := client.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return
				}

				var res statusResponse
				if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
					return
				}

				if res.DeviceID == "" || res.DeviceID == myID {
					return
				}

				mu.Lock()
				peers = append(peers, DiscoveredPeer{
					ID:           res.DeviceID,
					Name:         res.DeviceName,
					Addr:         targetAddr,
					Port:         port,
					LastSeen:     time.Now(),
					ScreenWidth:  res.ScreenWidth,
					ScreenHeight: res.ScreenHeight,
				})
				mu.Unlock()
			}(ip)
		}
	}

	wg.Wait()
	return peers
}

func getLocalSubnetPrefixes() []string {
	var prefixes []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return prefixes
	}

	for _, iface := range ifaces {
		// Skip loopback or down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}

			// Exclude common virtual/docker interfaces if possible, but keep private ranges
			if ip4[0] == 10 || (ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || (ip4[0] == 192 && ip4[1] == 168) {
				prefix := fmt.Sprintf("%d.%d.%d", ip4[0], ip4[1], ip4[2])
				prefixes = append(prefixes, prefix)
			}
		}
	}

	return prefixes
}

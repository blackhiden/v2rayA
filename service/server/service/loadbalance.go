package service

import (
	"sync"

	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

// loadBalancer manages round-robin state for outbound groups
type loadBalancer struct {
	mu      sync.Mutex
	indexes map[string]int // outbound name -> current index in server list
}

var lb = &loadBalancer{
	indexes: make(map[string]int),
}

// GetNextServerIndex returns the next server index for the given outbound group
// when load balancing is enabled. It uses round-robin selection.
func GetNextServerIndex(outbound string, serverCount int) int {
	if serverCount <= 0 {
		return 0
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Get current index for this outbound, default to 0
	idx := lb.indexes[outbound]
	// Increment for next time
	lb.indexes[outbound] = (idx + 1) % serverCount

	return idx
}

// ResetIndex resets the round-robin index for a specific outbound group
// Call this when connections change or when load balancing is disabled
func ResetIndex(outbound string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	delete(lb.indexes, outbound)
}

// GetServerIndices returns all current indices (for debugging/testing)
func GetServerIndices() map[string]int {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	result := make(map[string]int)
	for k, v := range lb.indexes {
		result[k] = v
	}
	return result
}

// SelectLoadBalancedServers selects servers for an outbound group based on
// load balancing settings. When load balancing is enabled and multiple servers
// exist in the group, it returns all servers (the balancer will distribute).
// When disabled, it returns only the first server in the group.
func SelectLoadBalancedServers(outbound string, connectedServers []*configure.Which) ([]*configure.Which, error) {
	setting := GetSetting()

	// If load balancing is disabled, return only the first server
	if !setting.LoadBalance {
		if len(connectedServers) > 0 {
			return []*configure.Which{connectedServers[0]}, nil
		}
		return []*configure.Which{}, nil
	}

	// Load balancing enabled - return all connected servers for this outbound
	// The v2ray core's balancer will handle round-robin distribution
	var result []*configure.Which
	for _, cs := range connectedServers {
		if cs.Outbound == outbound {
			result = append(result, cs)
		}
	}
	return result, nil
}

// ApplyLoadBalancing applies load balancing logic to the connected servers.
// When load balancing is enabled and there are multiple servers in an outbound group,
// this ensures the configuration is set up for the balancer to distribute connections.
func ApplyLoadBalancing() error {
	setting := GetSetting()

	if !setting.LoadBalance {
		return nil
	}

	// Get all connected servers
	allConnected := configure.GetConnectedServers()
	if allConnected == nil || allConnected.Len() == 0 {
		return nil
	}

	// Group servers by outbound
	outboundServers := make(map[string][]*configure.Which)
	for _, cs := range allConnected.Get() {
		outbound := cs.Outbound
		if outbound == "" {
			outbound = "proxy"
		}
		outboundServers[outbound] = append(outboundServers[outbound], cs)
	}

	// For each outbound group with multiple servers, we keep all servers
	// The balancer in v2ray core will handle the distribution
	// No need to modify connections - just ensure config is regenerated
	log.Debug("Load balancing enabled for outbounds: %v", outboundServers)

	return nil
}
package configure

import (
	"fmt"
	"strings"
)

type ObservatoryType string

func (t ObservatoryType) String() string {
	return string(t)
}

const (
	LeastPing ObservatoryType = "leastping"

	LoadBalanceStrategyRoundRobin = "roundrobin"
)

type OutboundSetting struct {
	ProbeURL            string          `json:"probeURL"`
	ProbeInterval       string          `json:"probeInterval"`
	Type                ObservatoryType `json:"type"`
	// Selected is the share link of the member the group routes through
	// alone; empty means every member, balanced by the observatory. A link
	// that matches no member is ignored, so a removed or renamed node
	// falls back to balancing instead of breaking the group.
	Selected            string          `json:"selected,omitempty"`
	LoadBalanceEnabled  bool            `json:"loadBalanceEnabled"`
	LoadBalanceStrategy string          `json:"loadBalanceStrategy"`
}

// DefaultOutboundSetting returns an OutboundSetting with default values.
func DefaultOutboundSetting() OutboundSetting {
	return OutboundSetting{
		ProbeURL:            DefaultProbeURL,
		ProbeInterval:       DefaultProbeInterval,
		Type:                ObservatoryType(DefaultOutboundType),
		LoadBalanceStrategy: LoadBalanceStrategyRoundRobin,
	}
}

// NormalizeLoadBalanceStrategy applies the default and validates strategies
// that the current Xray configuration generator supports.
func NormalizeLoadBalanceStrategy(strategy string) (string, error) {
	strategy = strings.ToLower(strings.TrimSpace(strategy))
	if strategy == "" {
		strategy = LoadBalanceStrategyRoundRobin
	}
	if strategy != LoadBalanceStrategyRoundRobin {
		return "", fmt.Errorf("unsupported load balancing strategy: %q", strategy)
	}
	return strategy, nil
}

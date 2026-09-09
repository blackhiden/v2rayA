package configure

import "testing"

func TestDefaultOutboundSettingLoadBalance(t *testing.T) {
	setting := DefaultOutboundSetting()
	if setting.LoadBalanceEnabled {
		t.Fatal("load balancing must be disabled by default")
	}
	if setting.LoadBalanceStrategy != LoadBalanceStrategyRoundRobin {
		t.Fatalf("expected default strategy %q, got %q", LoadBalanceStrategyRoundRobin, setting.LoadBalanceStrategy)
	}
}

func TestNormalizeLoadBalanceStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "", want: LoadBalanceStrategyRoundRobin, ok: true},
		{input: " ROUNDROBIN ", want: LoadBalanceStrategyRoundRobin, ok: true},
		{input: "leastping", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeLoadBalanceStrategy(tt.input)
			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected an error")
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

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

func TestNormalizeActiveOutboundName(t *testing.T) {
	outbounds := []string{DefaultOutboundName, "fast", "backup"}
	tests := []struct {
		name string
		want string
	}{
		{name: DefaultOutboundName, want: DefaultOutboundName},
		{name: "fast", want: "fast"},
		{name: "removed", want: DefaultOutboundName},
		{name: "", want: DefaultOutboundName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeActiveOutboundName(test.name, outbounds); got != test.want {
				t.Fatalf("normalizeActiveOutboundName(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

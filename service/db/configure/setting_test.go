package configure

import (
	"testing"
)

func TestSettingDefaultValues(t *testing.T) {
	setting := NewSetting()

	if setting.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got '%s'", setting.LogLevel)
	}

	if setting.RulePortMode != WhitelistMode {
		t.Errorf("Expected RulePortMode to be WhitelistMode, got '%v'", setting.RulePortMode)
	}

	if setting.Transparent != TransparentClose {
		t.Errorf("Expected Transparent to be TransparentClose, got '%v'", setting.Transparent)
	}

	if setting.TunAutoRoute != true {
		t.Errorf("Expected TunAutoRoute to be true, got '%v'", setting.TunAutoRoute)
	}

	// Load Balancing should be disabled by default
	if setting.LoadBalance != false {
		t.Errorf("Expected LoadBalance to be false by default, got '%v'", setting.LoadBalance)
	}

	// DNS defaults should be set
	if setting.DnsListenAddr != "0.0.0.0:52353" {
		t.Errorf("Expected DnsListenAddr to be '0.0.0.0:52353', got '%s'", setting.DnsListenAddr)
	}

	if setting.DnsCacheSize != 4096 {
		t.Errorf("Expected DnsCacheSize to be 4096, got '%d'", setting.DnsCacheSize)
	}
}

func TestSettingLoadBalanceToggle(t *testing.T) {
	tests := []struct {
		name      string
		loadBalance bool
		expect    bool
	}{
		{"LoadBalance disabled", false, false},
		{"LoadBalance enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting := NewSetting()
			setting.LoadBalance = tt.loadBalance

			if setting.LoadBalance != tt.expect {
				t.Errorf("Expected LoadBalance to be %v, got %v", tt.expect, setting.LoadBalance)
			}
		})
	}
}

func TestSettingMigrateLoadBalance(t *testing.T) {
	// Test that old configs without LoadBalance field get default value
	tests := []struct {
		name           string
		existingSetting *Setting
		expectLoadBalance bool
	}{
		{
			name: "Old config without LoadBalance field",
			existingSetting: &Setting{
				LogLevel: "info",
				Transparent: TransparentClose,
			},
			expectLoadBalance: false,
		},
		{
			name: "Config with LoadBalance explicitly set",
			existingSetting: &Setting{
				LogLevel: "info",
				Transparent: TransparentClose,
				LoadBalance: true,
			},
			expectLoadBalance: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting := NewSetting()
			setting.LoadBalance = tt.existingSetting.LoadBalance

			if setting.LoadBalance != tt.expectLoadBalance {
				t.Errorf("Expected LoadBalance to be %v, got %v", tt.expectLoadBalance, setting.LoadBalance)
			}
		})
	}
}
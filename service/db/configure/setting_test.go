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

	// DNS defaults should be set
	if setting.DnsListenAddr != "0.0.0.0:52353" {
		t.Errorf("Expected DnsListenAddr to be '0.0.0.0:52353', got '%s'", setting.DnsListenAddr)
	}

	if setting.DnsCacheSize != 4096 {
		t.Errorf("Expected DnsCacheSize to be 4096, got '%d'", setting.DnsCacheSize)
	}
}

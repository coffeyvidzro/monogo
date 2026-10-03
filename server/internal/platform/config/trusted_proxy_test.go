package config

import "testing"

func TestTrustedProxyPrefixes(t *testing.T) {
	cfg := Config{
		TrustedProxyCIDRs: []string{
			"203.0.113.42/24",
		},
	}
	prefixes, err := cfg.TrustedProxyPrefixes()
	if err != nil {
		t.Fatalf("TrustedProxyPrefixes() error = %v", err)
	}
	if len(prefixes) != 1 || prefixes[0].String() != "203.0.113.0/24" {
		t.Fatalf("TrustedProxyPrefixes() = %v", prefixes)
	}
	cfg.TrustedProxyCIDRs = []string{
		"invalid",
	}
	if _, err := cfg.TrustedProxyPrefixes(); err == nil {
		t.Fatal("TrustedProxyPrefixes(invalid) error = nil")
	}
}

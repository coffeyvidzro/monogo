package httpclient

import (
	"context"
	"net/netip"
	"testing"
)

func TestPublicTransportRejectsInternalDestinations(t *testing.T) {
	for _, address := range []string{
		"127.0.0.1:443", "10.0.0.1:443", "172.29.0.20:443",
		"192.168.1.1:443", "169.254.169.254:443", "100.64.0.1:443",
		"[::1]:443", "[::ffff:127.0.0.1]:443", "[fc00::1]:443",
		"[fe80::1]:443", "localhost:443",
	} {
		t.Run(address, func(t *testing.T) {
			conn, err := PublicTransport().DialContext(t.Context(), "tcp", address)
			if conn != nil {
				_ = conn.Close()
			}
			if err == nil {
				t.Fatal("internal destination accepted")
			}
		})
	}
}

func TestTransportDoesNotDelegateDestinationChecksToProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://8.8.8.8:3128")
	if PublicTransport().Proxy != nil {
		t.Fatal("environment proxy can bypass destination checks")
	}
}

func TestPublicAddressAndCancelledResolution(t *testing.T) {
	if err := ValidateAddress(netip.MustParseAddr("8.8.8.8")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := resolveHost(ctx, "unresolvable.invalid"); err == nil {
		t.Fatal("cancelled DNS resolution succeeded")
	}
}

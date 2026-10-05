package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
)

// PublicTransport rejects internal destinations and dials the checked IP directly.
func PublicTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A proxy could resolve the destination without these address checks.
	transport.Proxy = nil
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = func(
		ctx context.Context,
		network, address string,
	) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("split webhook endpoint address: %w", err)
		}
		addresses, err := resolveHost(ctx, host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range addresses {
			conn, dialErr := dialer.DialContext(
				ctx,
				network,
				net.JoinHostPort(ip.String(), port),
			)
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		if lastErr == nil {
			lastErr = fmt.Errorf(
				"webhook endpoint resolved to no dialable addresses",
			)
		}
		return nil, fmt.Errorf("dial webhook endpoint: %w", lastErr)
	}
	transport.MaxIdleConns = 20
	transport.MaxIdleConnsPerHost = 2
	transport.IdleConnTimeout = 30 * time.Second
	transport.ResponseHeaderTimeout = 10 * time.Second
	return transport
}

func resolveHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if parsed, err := netip.ParseAddr(host); err == nil {
		if err := ValidateAddress(parsed); err != nil {
			return nil, err
		}
		return []netip.Addr{parsed}, nil
	}

	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve webhook endpoint: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("webhook endpoint resolved to no addresses")
	}
	for _, address := range addresses {
		if err := ValidateAddress(address); err != nil {
			return nil, err
		}
	}
	return addresses, nil
}

func ValidateAddress(address netip.Addr) error {
	address = address.Unmap()
	if !address.IsValid() ||
		!address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() ||
		address.IsMulticast() ||
		address.IsUnspecified() ||
		isCarrierGradeNAT(address) {
		return fmt.Errorf("webhook endpoint resolves to a non-public address")
	}
	return nil
}

func isCarrierGradeNAT(address netip.Addr) bool {
	if !address.Is4() {
		return false
	}
	prefix := netip.MustParsePrefix("100.64.0.0/10")
	return prefix.Contains(address)
}

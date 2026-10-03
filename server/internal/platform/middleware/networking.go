package middleware

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/google/uuid"
)

type networkPolicyEvaluator interface {
	Allows(context.Context, uuid.UUID, netip.Addr) (bool, error)
}

type NetworkingMiddleware struct {
	policies       networkPolicyEvaluator
	trustedProxies []netip.Prefix
}

func NewNetworkingMiddleware(policies networkPolicyEvaluator, trustedProxies []netip.Prefix) *NetworkingMiddleware {
	return &NetworkingMiddleware{
		policies:       policies,
		trustedProxies: append([]netip.Prefix(nil), trustedProxies...),
	}
}

func (m *NetworkingMiddleware) Enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		organizationID, ok := OrganizationIDFromContext(r.Context())
		if !ok {
			httputil.Error(w, apperror.NewBadRequest("organization context required"))
			return
		}
		address, err := clientAddress(r, m.trustedProxies)
		if err != nil {
			httputil.Error(w, apperror.NewForbidden("request source address is unavailable"))
			return
		}
		allowed, err := m.policies.Allows(r.Context(), organizationID, address)
		if err != nil {
			httputil.Error(w, err)
			return
		}
		if !allowed {
			httputil.Error(w, apperror.NewForbidden("request source is blocked by organization network policy"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientAddress(r *http.Request, trustedProxies []netip.Prefix) (netip.Addr, error) {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, err
	}
	peer = peer.Unmap()
	if !containsAddress(trustedProxies, peer) {
		return peer, nil
	}
	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for index := len(forwarded) - 1; index >= 0; index-- {
		candidate, parseErr := netip.ParseAddr(strings.TrimSpace(forwarded[index]))
		if parseErr != nil {
			return netip.Addr{}, parseErr
		}
		candidate = candidate.Unmap()
		if !containsAddress(trustedProxies, candidate) {
			return candidate, nil
		}
	}
	return peer, nil
}

func containsAddress(prefixes []netip.Prefix, address netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

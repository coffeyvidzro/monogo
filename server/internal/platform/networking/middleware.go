package networking

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/platform/entitlements"
	platformmiddleware "github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/google/uuid"
)

type entitlementChecker interface {
	Enabled(context.Context, uuid.UUID, entitlements.Capability) (bool, error)
}

type policyLister interface {
	List(context.Context, uuid.UUID) ([]Policy, error)
}

type Middleware struct {
	entitlements   entitlementChecker
	policies       policyLister
	trustedProxies []netip.Prefix
}

func NewMiddleware(entitlementService entitlementChecker, policies policyLister, trustedProxies []netip.Prefix) *Middleware {
	return &Middleware{
		entitlements:   entitlementService,
		policies:       policies,
		trustedProxies: append([]netip.Prefix(nil), trustedProxies...),
	}
}

func (m *Middleware) Enforce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		organizationID, ok := platformmiddleware.OrganizationIDFromContext(r.Context())
		if !ok {
			httputil.Error(w, apperror.NewBadRequest("organization context required"))
			return
		}
		enabled, err := m.entitlements.Enabled(r.Context(), organizationID, entitlements.CapabilityPrivateNetworking)
		if err != nil {
			httputil.Error(w, err)
			return
		}
		if !enabled {
			next.ServeHTTP(w, r)
			return
		}
		address, err := clientAddress(r, m.trustedProxies)
		if err != nil {
			httputil.Error(w, apperror.NewForbidden("request source address is unavailable"))
			return
		}
		policies, err := m.policies.List(r.Context(), organizationID)
		if err != nil {
			httputil.Error(w, err)
			return
		}
		if !Allows(policies, address) {
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
	if !containsAddress(trustedProxies, peer) {
		return peer.Unmap(), nil
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
	return peer.Unmap(), nil
}

func containsAddress(prefixes []netip.Prefix, address netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

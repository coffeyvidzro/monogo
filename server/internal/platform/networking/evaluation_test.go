package networking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/platform/entitlements"
	platformmiddleware "github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/google/uuid"
)

func TestAllowsUsesDenyPrecedenceAndAllowDefault(t *testing.T) {
	address := netip.MustParseAddr("192.0.2.10")
	policies := []Policy{
		{
			Action:     ActionAllow,
			Status:     StatusActive,
			SourceCIDR: netip.MustParsePrefix("192.0.2.0/24"),
		},
		{
			Action:     ActionDeny,
			Status:     StatusActive,
			SourceCIDR: netip.MustParsePrefix("192.0.2.10/32"),
		},
	}
	if Allows(policies, address) {
		t.Fatal("Allows() = true for matching deny")
	}
	if Allows(policies[:1], netip.MustParseAddr("198.51.100.1")) {
		t.Fatal("Allows() = true for unmatched address with active allow list")
	}
	if !Allows(nil, address) {
		t.Fatal("Allows() = false without policies")
	}
}

func TestClientAddressTrustsOnlyConfiguredProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.9:443"
	request.Header.Set("X-Forwarded-For", "192.0.2.20")
	address, err := clientAddress(request, nil)
	if err != nil || address.String() != "203.0.113.9" {
		t.Fatalf("clientAddress() = (%q, %v)", address, err)
	}
	address, err = clientAddress(request, []netip.Prefix{
		netip.MustParsePrefix("203.0.113.0/24"),
	})
	if err != nil || address.String() != "192.0.2.20" {
		t.Fatalf("trusted clientAddress() = (%q, %v)", address, err)
	}
}

type entitlementStub struct{}

func (entitlementStub) Enabled(context.Context, uuid.UUID, entitlements.Capability) (bool, error) {
	return true, nil
}

type policyStub struct{ organizationID uuid.UUID }

func (s *policyStub) List(_ context.Context, organizationID uuid.UUID) ([]Policy, error) {
	s.organizationID = organizationID
	return nil, nil
}

func TestMiddlewareUsesAuthenticatedOrganization(t *testing.T) {
	organizationID := uuid.New()
	policies := &policyStub{}
	network := NewMiddleware(entitlementStub{}, policies, nil)
	organization := platformmiddleware.NewOrganizationMiddleware()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request = request.WithContext(authn.WithPrincipal(request.Context(), authn.Principal{
		Subject: authn.Subject{
			ID:   uuid.New(),
			Type: authn.SubjectOrganizationToken,
		},
		Credential: authn.Credential{
			Type: authn.CredentialOrganizationToken,
		},
		OrganizationID: organizationID,
	}))
	recorder := httptest.NewRecorder()
	handler := organization.Require(network.Enforce(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if policies.organizationID != organizationID {
		t.Fatalf("policy organization = %s, want %s", policies.organizationID, organizationID)
	}
}

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/google/uuid"
)

type networkPolicyEvaluatorStub struct {
	organizationID uuid.UUID
	address        netip.Addr
	allowed        bool
}

func (s *networkPolicyEvaluatorStub) Allows(_ context.Context, organizationID uuid.UUID, address netip.Addr) (bool, error) {
	s.organizationID = organizationID
	s.address = address
	return s.allowed, nil
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

func TestNetworkingMiddlewareUsesAuthenticatedOrganization(t *testing.T) {
	organizationID := uuid.New()
	evaluator := &networkPolicyEvaluatorStub{allowed: true}
	network := NewNetworkingMiddleware(evaluator, nil)
	organization := NewOrganizationMiddleware()
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
	if evaluator.organizationID != organizationID {
		t.Fatalf("policy organization = %s, want %s", evaluator.organizationID, organizationID)
	}
	if evaluator.address.String() != "192.0.2.10" {
		t.Fatalf("policy address = %s", evaluator.address)
	}
}

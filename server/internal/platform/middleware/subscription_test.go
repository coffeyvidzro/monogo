package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type subscriptionEntitlementStub struct {
	err            error
	organizationID uuid.UUID
}

func (s *subscriptionEntitlementStub) RequireActive(
	_ context.Context,
	organizationID uuid.UUID,
) error {
	s.organizationID = organizationID

	return s.err
}

func TestRequireSubscriptionAllowsActiveOrganization(t *testing.T) {
	organizationID := uuid.New()
	service := &subscriptionEntitlementStub{}
	nextCalled := false
	handler := RequireSubscription(service)(http.HandlerFunc(func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(
		withOrganizationContext(
			context.Background(),
			organizationContext{
				ID: organizationID,
			},
		),
		http.MethodGet,
		"/",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if service.organizationID != organizationID {
		t.Fatalf("organization id = %s, want %s", service.organizationID, organizationID)
	}
}

func TestRequireSubscriptionRejectsInactiveOrganization(t *testing.T) {
	organizationID := uuid.New()
	service := &subscriptionEntitlementStub{
		err: apperror.NewPaymentRequired("active subscription required"),
	}
	handler := RequireSubscription(service)(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		t.Fatal("next handler must not be called")
	}))
	request := httptest.NewRequestWithContext(
		withOrganizationContext(
			context.Background(),
			organizationContext{
				ID: organizationID,
			},
		),
		http.MethodPost,
		"/",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusPaymentRequired)
	}
}

func TestRequireSubscriptionRejectsMissingOrganizationContext(t *testing.T) {
	service := &subscriptionEntitlementStub{}
	handler := RequireSubscription(service)(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		t.Fatal("next handler must not be called")
	}))
	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if service.organizationID != uuid.Nil {
		t.Fatalf("subscription service called with organization %s", service.organizationID)
	}
}

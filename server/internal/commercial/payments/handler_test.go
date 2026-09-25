package payments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type stubPaymentService struct {
	row                                   sqlc.Payment
	organizationID, checkoutID, paymentID uuid.UUID
	provider, key                         string
}

func (s *stubPaymentService) CreateForCheckout(_ context.Context, organizationID, checkoutID uuid.UUID, provider, key string) (sqlc.Payment, error) {
	s.organizationID, s.checkoutID, s.provider, s.key = organizationID, checkoutID, provider, key
	return s.row, nil
}
func (s *stubPaymentService) Get(_ context.Context, organizationID, paymentID uuid.UUID) (sqlc.Payment, error) {
	s.organizationID, s.paymentID = organizationID, paymentID
	return s.row, nil
}

func TestPaymentRoutes(t *testing.T) {
	organizationID, checkoutID, paymentID := uuid.New(), uuid.New(), uuid.New()
	service := &stubPaymentService{row: sqlc.Payment{ID: paymentID, CheckoutID: checkoutID, Provider: "stripe", AmountMinor: 500, Currency: "USD", Status: "created"}}
	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(service), paymentAuth(organizationID), func(next http.Handler) http.Handler { return next })

	request := httptest.NewRequest(http.MethodPost, "/checkouts/"+checkoutID.String()+"/payments", strings.NewReader(`{"provider":"stripe"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "payment-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || service.provider != "stripe" || service.key != "payment-1" || service.checkoutID != checkoutID {
		t.Fatalf("create status=%d service=%+v", response.Code, service)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/payments/"+paymentID.String(), nil))
	if response.Code != http.StatusOK || service.organizationID != organizationID || service.paymentID != paymentID {
		t.Fatalf("get status=%d service=%+v", response.Code, service)
	}
}

func paymentAuth(organizationID uuid.UUID) func(http.Handler) http.Handler {
	organization := middleware.NewOrganizationMiddleware()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := authn.Principal{Subject: authn.Subject{ID: uuid.New(), Type: authn.SubjectOrganizationToken}, Credential: authn.Credential{Type: authn.CredentialOrganizationToken}, OrganizationID: organizationID}
			organization.Require(next).ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), principal)))
		})
	}
}

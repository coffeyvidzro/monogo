package checkout

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type stubCheckoutService struct {
	row                        sqlc.Checkout
	request                    CreateRequest
	organizationID, checkoutID uuid.UUID
}

func (s *stubCheckoutService) Create(_ context.Context, request CreateRequest) (sqlc.Checkout, error) {
	s.request = request
	return s.row, nil
}
func (s *stubCheckoutService) Get(_ context.Context, organizationID, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	s.organizationID, s.checkoutID = organizationID, checkoutID
	return s.row, nil
}
func (s *stubCheckoutService) Cancel(_ context.Context, organizationID, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	s.organizationID, s.checkoutID = organizationID, checkoutID
	s.row.Status = "canceled"
	return s.row, nil
}

func TestCheckoutRoutes(t *testing.T) {
	organizationID, checkoutID, walletID := uuid.New(), uuid.New(), uuid.New()
	service := &stubCheckoutService{row: sqlc.Checkout{ID: checkoutID, WalletID: walletID, AmountMinor: 500, Currency: "USD", Status: "pending"}}
	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(service), checkoutAuth(organizationID), func(next http.Handler) http.Handler { return next })

	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	create := httptest.NewRequest(http.MethodPost, "/checkouts", strings.NewReader(`{"wallet_id":"`+walletID.String()+`","amount_minor":500,"currency":"USD","expires_at":"`+expiresAt+`"}`))
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Idempotency-Key", "checkout-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, create)
	if response.Code != http.StatusCreated || service.request.OrganizationID != organizationID || service.request.IdempotencyKey != "checkout-1" {
		t.Fatalf("create status=%d request=%+v", response.Code, service.request)
	}

	for _, tc := range []struct{ method, target string }{{http.MethodGet, "/checkouts/" + checkoutID.String()}, {http.MethodPost, "/checkouts/" + checkoutID.String() + "/cancel"}} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.target, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s: status=%d body=%s", tc.method, tc.target, response.Code, response.Body.String())
		}
	}
	if service.organizationID != organizationID || service.checkoutID != checkoutID {
		t.Fatalf("unexpected IDs: %+v", service)
	}
}

func checkoutAuth(organizationID uuid.UUID) func(http.Handler) http.Handler {
	organization := middleware.NewOrganizationMiddleware()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := authn.Principal{Subject: authn.Subject{ID: uuid.New(), Type: authn.SubjectOrganizationToken}, Credential: authn.Credential{Type: authn.CredentialOrganizationToken}, OrganizationID: organizationID}
			organization.Require(next).ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), principal)))
		})
	}
}

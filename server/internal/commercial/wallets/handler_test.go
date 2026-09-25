package wallets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type stubReader struct {
	organizationID uuid.UUID
	walletID       uuid.UUID
	limit          int32
}

func (s *stubReader) List(_ context.Context, organizationID uuid.UUID) ([]sqlc.Wallet, error) {
	s.organizationID = organizationID
	return []sqlc.Wallet{{ID: s.walletID, Currency: "USD", BalanceMinor: 1250}}, nil
}

func (s *stubReader) GetByID(_ context.Context, organizationID, walletID uuid.UUID) (sqlc.Wallet, error) {
	s.organizationID, s.walletID = organizationID, walletID
	return sqlc.Wallet{ID: walletID, Currency: "USD", BalanceMinor: 1250}, nil
}

func (s *stubReader) ListTransactions(_ context.Context, organizationID, walletID uuid.UUID, limit int32) ([]sqlc.WalletTransaction, error) {
	s.organizationID, s.walletID, s.limit = organizationID, walletID, limit
	return []sqlc.WalletTransaction{{ID: uuid.New(), WalletID: walletID, Direction: "credit", Reason: "topup", AmountMinor: 1250, BalanceAfterMinor: 1250, ReferenceType: "payment", ReferenceID: uuid.New()}}, nil
}

func TestWalletRoutes(t *testing.T) {
	organizationID, walletID := uuid.New(), uuid.New()
	service := &stubReader{walletID: walletID}
	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(service), organizationAuth(organizationID))

	tests := []struct{ name, target, key string }{
		{name: "list", target: "/wallets", key: "wallets"},
		{name: "get", target: "/wallets/" + walletID.String(), key: "currency"},
		{name: "transactions", target: "/wallets/" + walletID.String() + "/transactions?limit=25", key: "transactions"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.target, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			data, ok := body["data"].(map[string]any)
			if !ok {
				t.Fatalf("missing data envelope: %#v", body)
			}
			if _, ok := data[tc.key]; !ok {
				t.Fatalf("missing %q: %#v", tc.key, data)
			}
		})
	}
	if service.organizationID != organizationID || service.walletID != walletID || service.limit != 25 {
		t.Fatalf("unexpected service arguments: %+v", service)
	}
}

func TestWalletRoutesRejectInvalidInput(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(&stubReader{}), organizationAuth(uuid.New()))
	for _, target := range []string{"/wallets/not-a-uuid", "/wallets/" + uuid.NewString() + "/transactions?limit=201"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", target, response.Code)
		}
	}
}

func organizationAuth(organizationID uuid.UUID) func(http.Handler) http.Handler {
	organization := middleware.NewOrganizationMiddleware()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal := authn.Principal{Subject: authn.Subject{ID: uuid.New(), Type: authn.SubjectOrganizationToken}, Credential: authn.Credential{Type: authn.CredentialOrganizationToken}, OrganizationID: organizationID}
			organization.Require(next).ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), principal)))
		})
	}
}

package scim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type authenticatorStub struct {
	secret    string
	principal Principal
}

func (s *authenticatorStub) Authenticate(_ context.Context, secret string) (Principal, error) {
	s.secret = secret
	return s.principal, nil
}
func TestRequireTokenEstablishesSCIMPrincipal(t *testing.T) {
	want := Principal{
		TokenID:        uuid.New(),
		OrganizationID: uuid.New(),
	}
	authenticator := &authenticatorStub{
		principal: want,
	}
	middleware := NewMiddleware(authenticator)
	request := httptest.NewRequest(http.MethodGet, "/scim/v2/Users", nil)
	request.Header.Set("Authorization", "Bearer lm_scim_secret")
	recorder := httptest.NewRecorder()
	handler := middleware.RequireToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := PrincipalFromContext(r.Context())
		if !ok || got != want {
			t.Fatalf("PrincipalFromContext() = (%+v, %t)", got, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || authenticator.secret != "lm_scim_secret" {
		t.Fatalf("status = %d, secret = %q", recorder.Code, authenticator.secret)
	}
}
func TestRequireTokenRejectsMissingBearer(t *testing.T) {
	middleware := NewMiddleware(&authenticatorStub{})
	recorder := httptest.NewRecorder()
	middleware.RequireToken(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called")
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
}

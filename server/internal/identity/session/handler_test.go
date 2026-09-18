package session

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/security/authn"
)

func TestSetCookieIsHostOnlyInDevelopment(t *testing.T) {
	recorder := httptest.NewRecorder()

	SetCookie(recorder, "secret", time.Now().Add(time.Hour), true, "leamout.com")

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != authn.SessionCookieName {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, authn.SessionCookieName)
	}
	if cookie.Domain != "" {
		t.Fatalf("cookie domain = %q, want host-only cookie", cookie.Domain)
	}
	if cookie.Secure {
		t.Fatal("development cookie is secure, want false")
	}
}

func TestSetCookieUsesProductionDomain(t *testing.T) {
	recorder := httptest.NewRecorder()

	SetCookie(recorder, "secret", time.Now().Add(time.Hour), false, " leamout.com ")

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Domain != "leamout.com" {
		t.Fatalf("cookie domain = %q, want leamout.com", cookie.Domain)
	}
	if !cookie.Secure {
		t.Fatal("production cookie is not secure")
	}
}

func TestClearCookieUsesProductionDomain(t *testing.T) {
	recorder := httptest.NewRecorder()

	ClearCookie(recorder, false, "leamout.com")

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Domain != "leamout.com" {
		t.Fatalf("cookie domain = %q, want leamout.com", cookie.Domain)
	}
	if cookie.MaxAge >= 0 {
		t.Fatalf("cookie MaxAge = %d, want negative", cookie.MaxAge)
	}
	if !cookie.Secure {
		t.Fatal("production clear cookie is not secure")
	}
}

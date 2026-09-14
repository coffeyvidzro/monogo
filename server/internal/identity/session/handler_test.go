package session

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leamout/leamout/internal/security/authn"
)

func TestSetCookieIsHostOnlyByDefault(t *testing.T) {
	t.Setenv("LEAMOUT_SESSION_COOKIE_DOMAIN", "")
	recorder := httptest.NewRecorder()

	SetCookie(recorder, "secret", time.Now().Add(time.Hour))

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
}

func TestSetCookieUsesConfiguredSharedDomain(t *testing.T) {
	t.Setenv("LEAMOUT_SESSION_COOKIE_DOMAIN", "leamout.com")
	recorder := httptest.NewRecorder()

	SetCookie(recorder, "secret", time.Now().Add(time.Hour))

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	if domain := cookies[0].Domain; domain != "leamout.com" {
		t.Fatalf("cookie domain = %q, want leamout.com", domain)
	}
}

func TestClearCookieUsesConfiguredSharedDomain(t *testing.T) {
	t.Setenv("LEAMOUT_SESSION_COOKIE_DOMAIN", "leamout.com")
	recorder := httptest.NewRecorder()

	ClearCookie(recorder)

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
}

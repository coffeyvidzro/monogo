package realtime

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- verifies coturn's required HMAC-SHA1 REST credential.
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeIssueLimiter struct {
	allowed bool
	err     error
	key     string
	limit   int64
	window  time.Duration
}

func (f *fakeIssueLimiter) AllowFixedWindow(_ context.Context, key string, limit int64, window time.Duration) (bool, error) {
	f.key, f.limit, f.window = key, limit, window
	return f.allowed, f.err
}

func validTestConfig() Config {
	return Config{
		AuthSecret: "test-shared-secret-at-least-32-bytes",
		URLs:       []string{"turn:turn.example.com:3478"},
	}
}

func TestIssueCreatesTenantBoundExpiringCoturnCredential(t *testing.T) {
	limiter := &fakeIssueLimiter{allowed: true}
	service, err := NewService(Config{
		AuthSecret: "test-shared-secret-at-least-32-bytes",
		URLs:       []string{" stun:turn.example.com:3478 ", "turns:turn.example.com:5349?transport=tcp"},
	}, limiter)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	service.random = strings.NewReader("0123456789abcdef")
	organizationID := uuid.MustParse("70d9f72b-ff44-4ca0-8c26-5b782c011a5f")

	got, err := service.Issue(context.Background(), organizationID)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}
	if got.ExpiresAt != now.Add(10*time.Minute) {
		t.Fatalf("expires_at = %s, want %s", got.ExpiresAt, now.Add(10*time.Minute))
	}
	if len(got.ICEServers) != 1 {
		t.Fatalf("ICE servers = %d, want 1", len(got.ICEServers))
	}
	server := got.ICEServers[0]
	wantUsername := "1788178200:70d9f72b-ff44-4ca0-8c26-5b782c011a5f:30313233343536373839616263646566"
	if server.Username != wantUsername {
		t.Fatalf("username = %q, want %q", server.Username, wantUsername)
	}
	if server.URLs[0] != "stun:turn.example.com:3478" {
		t.Fatalf("trimmed URL = %q", server.URLs[0])
	}

	digest := hmac.New(sha1.New, []byte("test-shared-secret-at-least-32-bytes"))
	_, _ = digest.Write([]byte(wantUsername))
	wantCredential := base64.StdEncoding.EncodeToString(digest.Sum(nil))
	if server.Credential != wantCredential {
		t.Fatalf("credential = %q, want coturn HMAC %q", server.Credential, wantCredential)
	}
	if limiter.key != "ratelimit:turn-credentials:"+organizationID.String() || limiter.limit != 60 || limiter.window != time.Minute {
		t.Fatalf("unexpected limiter call: %+v", limiter)
	}
}

func TestIssueReturnsUniqueCredentials(t *testing.T) {
	service, err := NewService(validTestConfig(), &fakeIssueLimiter{allowed: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	service.random = strings.NewReader("0123456789abcdefFEDCBA9876543210")
	organizationID := uuid.New()

	first, err := service.Issue(context.Background(), organizationID)
	if err != nil {
		t.Fatalf("issue first credential: %v", err)
	}
	second, err := service.Issue(context.Background(), organizationID)
	if err != nil {
		t.Fatalf("issue second credential: %v", err)
	}
	if first.ICEServers[0].Username == second.ICEServers[0].Username {
		t.Fatal("successive TURN credentials reused a username")
	}
}

func TestNewServiceRejectsUnsafeConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{name: "missing secret", config: Config{URLs: []string{"turn:turn.example.com"}}},
		{name: "short secret", config: Config{AuthSecret: "secret", URLs: []string{"turn:turn.example.com"}}},
		{name: "missing URLs", config: Config{AuthSecret: strings.Repeat("s", 32)}},
		{name: "insecure URL scheme", config: Config{AuthSecret: strings.Repeat("s", 32), URLs: []string{"https://turn.example.com"}}},
		{name: "missing URL target", config: Config{AuthSecret: strings.Repeat("s", 32), URLs: []string{"turn:?transport=udp"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewService(tt.config, &fakeIssueLimiter{allowed: true}); err == nil {
				t.Fatal("NewService() error = nil")
			}
		})
	}
}

func TestIssueRejectsMissingOrganization(t *testing.T) {
	service, err := NewService(validTestConfig(), &fakeIssueLimiter{allowed: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := service.Issue(context.Background(), uuid.Nil); err == nil {
		t.Fatal("Issue() error = nil")
	}
}

func TestIssueFailsClosedWhenRateLimited(t *testing.T) {
	service, err := NewService(validTestConfig(), &fakeIssueLimiter{allowed: false})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := service.Issue(context.Background(), uuid.New()); !errors.Is(err, ErrIssueRateLimited) {
		t.Fatalf("Issue() error = %v, want %v", err, ErrIssueRateLimited)
	}
}

package transport

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenServiceIssueVerifyAndRejectReplay(t *testing.T) {
	service, err := NewTokenService(strings.Repeat("s", 32))
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	service.random = func(target []byte) (int, error) {
		for index := range target {
			target[index] = byte(index)
		}
		return len(target), nil
	}
	claims := TokenClaims{
		SessionID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(), OrganizationID: uuid.New(),
	}
	token, err := service.Issue(claims, time.Minute)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	got, err := service.VerifyAndConsume(token)
	if err != nil {
		t.Fatalf("VerifyAndConsume() error = %v", err)
	}
	if got.SessionID != claims.SessionID || got.ChannelID != claims.ChannelID {
		t.Fatalf("claims = %+v, want %+v", got, claims)
	}
	if _, err := service.VerifyAndConsume(token); err == nil {
		t.Fatal("VerifyAndConsume() accepted replayed token")
	}
}

func TestTokenServiceRejectsTamperingAndExpiry(t *testing.T) {
	service, err := NewTokenService(strings.Repeat("s", 32))
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	claims := TokenClaims{
		SessionID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(), OrganizationID: uuid.New(),
	}
	token, err := service.Issue(claims, time.Second)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := service.VerifyAndConsume(token + "x"); err == nil {
		t.Fatal("VerifyAndConsume() accepted tampered token")
	}
	service.now = func() time.Time { return now.Add(time.Second) }
	if _, err := service.VerifyAndConsume(token); err == nil {
		t.Fatal("VerifyAndConsume() accepted expired token")
	}
}

func TestNewTokenServiceRejectsShortSecret(t *testing.T) {
	if _, err := NewTokenService("short"); err == nil {
		t.Fatal("NewTokenService() error = nil")
	}
}

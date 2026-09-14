package session

import (
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

var (
	ErrInvalidSession = errors.New("invalid session")
)

func validateUserID(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("user id is required")
	}

	return nil
}

func validateSessionID(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("session id is required")
	}

	return nil
}

func validateToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidSession
	}

	return nil
}

func validateSessionExpiry(session sqlc.Session) error {
	if !session.ExpiresAt.Valid {
		return ErrInvalidSession
	}

	if !time.Now().Before(session.ExpiresAt.Time) {
		return ErrInvalidSession
	}

	return nil
}

func parseIP(value *string) *netip.Addr {
	if value == nil || *value == "" {
		return nil
	}

	ip, err := netip.ParseAddr(*value)
	if err != nil {
		return nil
	}

	return &ip
}

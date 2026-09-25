package checkout

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func validateCreate(req *CreateRequest) error {
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.OrganizationID == uuid.Nil || req.WalletID == uuid.Nil {
		return apperror.NewBadRequest("organization and wallet are required")
	}
	if len(req.Currency) != 3 {
		return apperror.NewBadRequest("checkout currency must be a three-letter code")
	}
	for _, ch := range req.Currency {
		if ch < 'A' || ch > 'Z' {
			return apperror.NewBadRequest("checkout currency must be a three-letter code")
		}
	}
	if req.AmountMinor <= 0 {
		return apperror.NewBadRequest("checkout amount must be positive")
	}
	if len(req.IdempotencyKey) == 0 || len(req.IdempotencyKey) > 255 {
		return apperror.NewBadRequest("checkout idempotency key is required")
	}
	if !req.ExpiresAt.After(time.Now()) {
		return apperror.NewBadRequest("checkout expiration must be in the future")
	}
	return nil
}

func requestHash(req CreateRequest) string {
	value := req.WalletID.String() + "\n" + req.Currency + "\n" +
		strconv.FormatInt(req.AmountMinor, 10) + "\n" +
		req.ExpiresAt.UTC().Format(time.RFC3339Nano)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

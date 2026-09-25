package lifecycle

import (
	"errors"
	"regexp"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var e164 = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
var postalCode = regexp.MustCompile(`^[A-Z0-9][A-Z0-9 -]{1,11}$`)

func validateLifecycleKey(organizationID uuid.UUID, key string) error {
	if organizationID == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	if len(key) < 1 || len(key) > 255 {
		return apperror.NewBadRequest("Idempotency-Key is required and must not exceed 255 characters")
	}
	return nil
}

func normalizeEmergency(req *EmergencyAddressRequest) {
	req.Name = strings.TrimSpace(req.Name)
	req.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	req.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	req.Locality = strings.TrimSpace(req.Locality)
	req.Region = strings.ToUpper(strings.TrimSpace(req.Region))
	req.PostalCode = strings.ToUpper(strings.TrimSpace(req.PostalCode))
	req.CountryCode = strings.ToUpper(strings.TrimSpace(req.CountryCode))
}

func validateEmergency(req EmergencyAddressRequest) error {
	if req.Name == "" || req.AddressLine1 == "" || req.Locality == "" || req.Region == "" {
		return apperror.NewBadRequest("complete emergency address is required")
	}
	if len(req.CountryCode) != 2 {
		return apperror.NewBadRequest("country_code must be a two-letter ISO country code")
	}
	if !postalCode.MatchString(req.PostalCode) {
		return apperror.NewBadRequest("postal_code is invalid")
	}
	return nil
}

func writeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("number or carrier connection not found")
	}
	var dbError *pgconn.PgError
	if errors.As(err, &dbError) {
		switch dbError.Code {
		case "23505":
			return apperror.NewConflict("number already exists")
		case "23503", "23514", "23502":
			return apperror.NewBadRequest("number or carrier connection is invalid")
		}
	}
	return apperror.NewInternal("update number", err)
}

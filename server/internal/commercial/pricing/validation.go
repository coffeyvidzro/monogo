package pricing

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var digitsPattern = regexp.MustCompile(`^[1-9][0-9]{0,14}$`)
var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

func normalizeCreateVoiceRateRequest(req *CreateVoiceRateRequest) error {
	if req.OrganizationID != nil && *req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is invalid", ErrInvalidInput)
	}

	req.DestinationPrefix = strings.TrimSpace(req.DestinationPrefix)
	if !digitsPattern.MatchString(req.DestinationPrefix) {
		return fmt.Errorf("%w: destination prefix is invalid", ErrInvalidInput)
	}

	direction, err := normalizeDirection(req.Direction)
	if err != nil {
		return err
	}
	req.Direction = direction

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("%w: currency must be a three-letter code", ErrInvalidInput)
	}
	if req.RateMicros <= 0 {
		return fmt.Errorf("%w: rate must be positive", ErrInvalidInput)
	}
	if req.EffectiveAt.IsZero() {
		return fmt.Errorf("%w: effective time is required", ErrInvalidInput)
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(req.EffectiveAt) {
		return fmt.Errorf("%w: expiry must follow effective time", ErrInvalidInput)
	}

	return nil
}

func normalizeResolveVoiceRateRequest(req *ResolveVoiceRateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}

	req.DestinationDigits = strings.TrimPrefix(
		strings.TrimSpace(req.DestinationDigits),
		"+",
	)
	if !digitsPattern.MatchString(req.DestinationDigits) {
		return fmt.Errorf("%w: destination is invalid", ErrInvalidInput)
	}

	direction, err := normalizeDirection(req.Direction)
	if err != nil {
		return err
	}
	req.Direction = direction

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("%w: currency must be a three-letter code", ErrInvalidInput)
	}

	return nil
}

func normalizeResolveProductRateRequest(req *ResolveProductRateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}

	req.Product = strings.TrimSpace(req.Product)
	switch req.Product {
	case ProductSMSOutbound,
		ProductWhatsAppOutbound,
		ProductNumberPurchase,
		ProductNumberRenewal:
	default:
		return fmt.Errorf("%w: product is invalid", ErrInvalidInput)
	}

	req.Selector = strings.ToUpper(strings.TrimSpace(req.Selector))
	if req.Selector != "*" && !countryCodePattern.MatchString(req.Selector) {
		return fmt.Errorf("%w: selector must be a country code or *", ErrInvalidInput)
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("%w: currency must be a three-letter code", ErrInvalidInput)
	}

	return nil
}

func normalizeDirection(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case DirectionInbound, DirectionOutbound:
		return value, nil
	default:
		return "", fmt.Errorf("%w: direction must be inbound or outbound", ErrInvalidInput)
	}
}

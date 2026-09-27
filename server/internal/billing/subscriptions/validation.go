package subscriptions

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func normalizePlan(req *CreatePlanRequest) error {
	req.Code = strings.ToLower(
		strings.TrimSpace(req.Code),
	)
	req.Name = strings.TrimSpace(
		req.Name,
	)
	req.Currency = strings.ToUpper(
		strings.TrimSpace(req.Currency),
	)

	if req.Code == "" || len(req.Code) > 64 {
		return apperror.NewBadRequest(
			"plan code must be between 1 and 64 characters",
		)
	}

	for index, character := range req.Code {
		validCharacter := (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '_'
		if !validCharacter {
			return apperror.NewBadRequest(
				"plan code must use lowercase letters, digits, or underscores",
			)
		}

		if index == 0 &&
			(character < 'a' || character > 'z') {
			return apperror.NewBadRequest(
				"plan code must start with a lowercase letter",
			)
		}
	}

	if req.Name == "" || len(req.Name) > 120 {
		return apperror.NewBadRequest(
			"plan name must be between 1 and 120 characters",
		)
	}

	if err := validateCurrency(req.Currency); err != nil {
		return err
	}

	if req.AmountMicros < 0 {
		return apperror.NewBadRequest(
			"plan amount cannot be negative",
		)
	}

	return nil
}

func normalizePlanCode(code string) (string, error) {
	code = strings.ToLower(
		strings.TrimSpace(code),
	)

	if code == "" {
		return "", apperror.NewBadRequest(
			"plan code is required",
		)
	}

	return code, nil
}

func validateCurrency(value string) error {
	if len(value) != 3 {
		return apperror.NewBadRequest(
			"currency must be a 3-letter ISO code",
		)
	}

	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return apperror.NewBadRequest(
				"currency must be a 3-letter ISO code",
			)
		}
	}

	return nil
}

func validateCreate(req CreateRequest) error {
	if req.OrganizationID == uuid.Nil ||
		req.PlanID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and plan are required",
		)
	}

	return nil
}

func validatePeriod(req PeriodRequest) error {
	if req.OrganizationID == uuid.Nil ||
		req.SubscriptionID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and subscription are required",
		)
	}

	if req.Start.IsZero() ||
		req.End.IsZero() ||
		!req.End.After(req.Start) {
		return apperror.NewBadRequest(
			"subscription period end must be after period start",
		)
	}

	return nil
}

func validateSubscriptionIdentity(
	organizationID uuid.UUID,
	subscriptionID uuid.UUID,
) error {
	if organizationID == uuid.Nil ||
		subscriptionID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and subscription are required",
		)
	}

	return nil
}

package credentials

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/security/authz"
)

const (
	maxNameLength        = 100
	maxDescriptionLength = 500
)

func ValidateCreate(input CreateInput) error {
	if input.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if input.CreatedBy == uuid.Nil {
		return fmt.Errorf("created by is required")
	}
	if err := validateName(input.Name); err != nil {
		return err
	}
	if input.Description != nil && len(*input.Description) > maxDescriptionLength {
		return fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	return ValidateScopes(input.Scopes)
}

func ValidateUpdate(input UpdateInput) error {
	if input.ID == uuid.Nil {
		return fmt.Errorf("credential id is required")
	}
	if input.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization id is required")
	}
	if input.Name != nil {
		if err := validateName(*input.Name); err != nil {
			return err
		}
	}
	if input.Description != nil && len(*input.Description) > maxDescriptionLength {
		return fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	if input.Scopes != nil {
		if err := ValidateScopes(*input.Scopes); err != nil {
			return err
		}
	}
	return nil
}

func ValidateScopes(scopes []string) error {
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if !authz.Scope(scope).IsValid() {
			return fmt.Errorf("invalid credential scope %q", scope)
		}
		if _, ok := seen[scope]; ok {
			return fmt.Errorf("duplicate credential scope %q", scope)
		}
		seen[scope] = struct{}{}
	}
	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("name must be at most %d characters", maxNameLength)
	}
	return nil
}

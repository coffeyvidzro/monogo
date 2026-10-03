package sso

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func normalizeCreate(req *CreateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Protocol = strings.ToLower(strings.TrimSpace(req.Protocol))
	req.Issuer = strings.TrimSpace(req.Issuer)
	if req.Secret != nil {
		value := strings.TrimSpace(*req.Secret)
		req.Secret = &value
	}
	if len(req.Configuration) == 0 {
		req.Configuration = json.RawMessage(`{}`)
	}
	return validateConnection(req.Name, req.Protocol, req.Issuer, req.Configuration, req.Secret, "")
}

func normalizeUpdate(req *UpdateRequest) error {
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		req.Name = &value
	}
	if req.Issuer != nil {
		value := strings.TrimSpace(*req.Issuer)
		req.Issuer = &value
	}
	if req.Secret != nil {
		value := strings.TrimSpace(*req.Secret)
		req.Secret = &value
	}
	if req.Status != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Status))
		req.Status = &value
	}
	if req.Name != nil && (*req.Name == "" || len(*req.Name) > 128) {
		return apperror.NewBadRequest("SSO connection name must be between 1 and 128 characters")
	}
	if req.Issuer != nil {
		if err := validateIssuer(*req.Issuer); err != nil {
			return err
		}
	}
	if req.Configuration != nil {
		if err := validateConfiguration(*req.Configuration); err != nil {
			return err
		}
	}
	if req.Secret != nil && *req.Secret == "" {
		return apperror.NewBadRequest("SSO secret cannot be empty")
	}
	if req.Status != nil && *req.Status != StatusActive && *req.Status != StatusDisabled {
		return apperror.NewBadRequest("SSO status must be active or disabled")
	}
	return nil
}

func validateConnection(name, protocol, issuer string, configuration json.RawMessage, secret *string, status string) error {
	if name == "" || len(name) > 128 {
		return apperror.NewBadRequest("SSO connection name must be between 1 and 128 characters")
	}
	if protocol != ProtocolSAML && protocol != ProtocolOIDC {
		return apperror.NewBadRequest("SSO protocol must be saml or oidc")
	}
	if err := validateIssuer(issuer); err != nil {
		return err
	}
	if err := validateConfiguration(configuration); err != nil {
		return err
	}
	if secret != nil && strings.TrimSpace(*secret) == "" {
		return apperror.NewBadRequest("SSO secret cannot be empty")
	}
	if status != "" && status != StatusActive && status != StatusDisabled {
		return apperror.NewBadRequest("SSO status must be active or disabled")
	}
	return nil
}

func validateIssuer(value string) error {
	if value == "" || len(value) > 2048 {
		return apperror.NewBadRequest("SSO issuer must be between 1 and 2048 characters")
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return apperror.NewBadRequest("SSO issuer must be an absolute URL")
	}
	if parsed.Scheme != "https" {
		return apperror.NewBadRequest("SSO issuer must use https")
	}
	return nil
}

func validateConfiguration(value json.RawMessage) error {
	var object map[string]any
	if err := json.Unmarshal(value, &object); err != nil || object == nil {
		return apperror.NewBadRequest("SSO configuration must be a JSON object")
	}
	return nil
}

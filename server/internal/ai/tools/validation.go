package tools

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func normalizeCreate(req CreateRequest) (CreateRequest, error) {
	if req.Type == "" { req.Type = TypeWebhook }
	req.Type = strings.TrimSpace(req.Type)
	if req.Type != TypeBuiltin && req.Type != TypeWebhook {
		return CreateRequest{}, apperror.NewBadRequest("tool type must be builtin or webhook")
	}
	var err error
	req.Name, err = required(req.Name, "name", 128); if err != nil { return CreateRequest{}, err }
	req.Description, err = required(req.Description, "description", 2000); if err != nil { return CreateRequest{}, err }
	if len(req.Parameters) == 0 { req.Parameters = json.RawMessage(`{}`) }
	if err := validateParameters(req.Parameters); err != nil { return CreateRequest{}, err }
	if req.TimeoutMS != nil && (*req.TimeoutMS < 100 || *req.TimeoutMS > 30000) {
		return CreateRequest{}, apperror.NewBadRequest("timeout_ms must be between 100 and 30000")
	}
	if err := validateEndpoint(req.Type, req.EndpointURL); err != nil { return CreateRequest{}, err }
	return req, nil
}

func normalizeUpdate(req UpdateRequest) (UpdateRequest, error) {
	if req.Name == nil && req.Description == nil && req.Parameters == nil && req.EndpointURL == nil && req.TimeoutMS == nil && req.Enabled == nil {
		return UpdateRequest{}, apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil { v, err := required(*req.Name, "name", 128); if err != nil { return UpdateRequest{}, err }; req.Name = &v }
	if req.Description != nil { v, err := required(*req.Description, "description", 2000); if err != nil { return UpdateRequest{}, err }; req.Description = &v }
	if req.Parameters != nil { if err := validateParameters(*req.Parameters); err != nil { return UpdateRequest{}, err } }
	if req.TimeoutMS != nil && (*req.TimeoutMS < 100 || *req.TimeoutMS > 30000) {
		return UpdateRequest{}, apperror.NewBadRequest("timeout_ms must be between 100 and 30000")
	}
	return req, nil
}

func validateParameters(value json.RawMessage) error {
	if !json.Valid(value) { return apperror.NewBadRequest("parameters must be valid JSON") }
	var object map[string]any
	if err := json.Unmarshal(value, &object); err != nil {
		return apperror.NewBadRequest("parameters must be a JSON object")
	}
	return nil
}

func validateEndpoint(toolType string, endpoint *string) error {
	if toolType == TypeBuiltin {
		if endpoint != nil { return apperror.NewBadRequest("builtin tools cannot define endpoint_url") }
		return nil
	}
	if endpoint == nil { return apperror.NewBadRequest("webhook tools require endpoint_url") }
	value := strings.TrimSpace(*endpoint)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return apperror.NewBadRequest("endpoint_url must be a valid HTTP or HTTPS URL")
	}
	*endpoint = value
	return nil
}

func required(value, field string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > max {
		return "", apperror.NewBadRequest(field + " must be between 1 and " + strconv.Itoa(max) + " characters")
	}
	return value, nil
}

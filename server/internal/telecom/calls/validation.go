package calls

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func validateOrganizationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	return nil
}

func validateCallID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("call id is required")
	}
	return nil
}

func validateIDs(organizationID, callID uuid.UUID) error {
	if err := validateOrganizationID(organizationID); err != nil {
		return err
	}
	return validateCallID(callID)
}

func normalizeCreateRequest(req CreateRequest) (CreateRequest, error) {
	req.Direction = DirectionOutbound
	req.FromURI = strings.TrimSpace(req.FromURI)
	req.ToURI = strings.TrimSpace(req.ToURI)
	req.DTMFMode = strings.ToLower(strings.TrimSpace(req.DTMFMode))
	req.MediaEncryption = strings.ToLower(strings.TrimSpace(req.MediaEncryption))

	if req.FromURI == "" {
		return CreateRequest{}, apperror.NewBadRequest("from_uri is required")
	}
	if req.ToURI == "" {
		return CreateRequest{}, apperror.NewBadRequest("to_uri is required")
	}
	if req.ApplicationID != nil && *req.ApplicationID == uuid.Nil {
		return CreateRequest{}, apperror.NewBadRequest("application_id is invalid")
	}
	if req.TrunkID != nil && *req.TrunkID == uuid.Nil {
		return CreateRequest{}, apperror.NewBadRequest("trunk_id is invalid")
	}
	switch req.DTMFMode {
	case "", "rfc2833", "info", "none":
	default:
		return CreateRequest{}, apperror.NewBadRequest("dtmf_mode is invalid")
	}
	switch req.MediaEncryption {
	case "", "none", "sdes_srtp":
	default:
		return CreateRequest{}, apperror.NewBadRequest("media_encryption is invalid")
	}
	return req, nil
}

func validateListRequest(req ListRequest) error {
	if req.Offset < 0 {
		return apperror.NewBadRequest("offset must not be negative")
	}
	if req.Limit < 1 || req.Limit > 200 {
		return apperror.NewBadRequest("limit must be between 1 and 200")
	}
	if req.State == nil {
		return nil
	}
	state := State(strings.TrimSpace(*req.State))
	switch state {
	case StateInitiating, StateRinging, StateAnswered, StateActive, StateCompleted, StateFailed, StateCancelled:
		return nil
	default:
		return apperror.NewBadRequest("invalid call state")
	}
}

func validateRouteAttribution(route RouteAttribution) error {
	if route.CarrierConnectionID == nil || *route.CarrierConnectionID == uuid.Nil {
		return apperror.NewBadRequest("carrier_connection_id is required")
	}
	if route.TrunkID != nil && *route.TrunkID == uuid.Nil {
		return apperror.NewBadRequest("trunk_id is invalid")
	}
	if route.TrunkEndpointID != nil && *route.TrunkEndpointID == uuid.Nil {
		return apperror.NewBadRequest("trunk_endpoint_id is invalid")
	}
	return nil
}

func normalizeOptionalReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	value := strings.TrimSpace(*reason)
	if value == "" {
		return nil
	}
	return &value
}

func normalizeTransfer(req TransferActionRequest) (TransferActionRequest, error) {
	req.Destination = strings.TrimSpace(req.Destination)
	if req.Destination == "" {
		return TransferActionRequest{}, apperror.NewBadRequest("destination is required")
	}
	return req, nil
}

func normalizePlay(req PlayActionRequest) (PlayActionRequest, error) {
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		return PlayActionRequest{}, apperror.NewBadRequest("path is required")
	}
	return req, nil
}

func normalizeRecord(req RecordActionRequest) (RecordActionRequest, error) {
	req.Path = strings.TrimSpace(req.Path)
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	if req.Path == "" {
		return RecordActionRequest{}, apperror.NewBadRequest("path is required")
	}
	if req.Action == "" {
		req.Action = "start"
	}
	if req.Action != "start" && req.Action != "stop" {
		return RecordActionRequest{}, apperror.NewBadRequest("action must be start or stop")
	}
	return req, nil
}

func normalizeDTMF(req DTMFActionRequest) (DTMFActionRequest, error) {
	req.Digits = strings.TrimSpace(req.Digits)
	if req.Digits == "" {
		return DTMFActionRequest{}, apperror.NewBadRequest("digits are required")
	}
	return req, nil
}

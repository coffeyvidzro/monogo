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
	switch req.Direction {
	case DirectionInbound, DirectionOutbound:
	default:
		return CreateRequest{}, apperror.NewBadRequest("direction must be inbound or outbound")
	}

	req.FromURI = strings.TrimSpace(req.FromURI)
	req.ToURI = strings.TrimSpace(req.ToURI)
	if req.FromURI == "" {
		return CreateRequest{}, apperror.NewBadRequest("from_uri is required")
	}
	if req.ToURI == "" {
		return CreateRequest{}, apperror.NewBadRequest("to_uri is required")
	}
	if req.ApplicationID != nil && *req.ApplicationID == uuid.Nil {
		return CreateRequest{}, apperror.NewBadRequest("application_id is invalid")
	}
	if req.SIPCallID != nil {
		value := strings.TrimSpace(*req.SIPCallID)
		if value == "" {
			return CreateRequest{}, apperror.NewBadRequest("sip_call_id cannot be empty")
		}
		req.SIPCallID = &value
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

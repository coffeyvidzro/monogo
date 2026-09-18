package routing

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func normalizeInboundRequest(req InboundRequest) InboundRequest {
	req.CalledNumber = strings.TrimSpace(req.CalledNumber)
	return req
}

func validateInboundRequest(req InboundRequest) error {
	if req.OrganizationID == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	if req.ApplicationID == uuid.Nil {
		return apperror.NewBadRequest("application_id is required")
	}
	if req.PhoneNumberID == uuid.Nil {
		return apperror.NewBadRequest("phone_number_id is required")
	}
	if req.CarrierConnectionID == uuid.Nil {
		return apperror.NewBadRequest("carrier_connection_id is required")
	}
	if req.CalledNumber == "" {
		return apperror.NewBadRequest("called_number is required")
	}
	return nil
}

func normalizeOutboundRequest(req OutboundRequest) OutboundRequest {
	req.Destination = strings.TrimSpace(req.Destination)
	return req
}

func validateOutboundRequest(req OutboundRequest) error {
	if req.OrganizationID == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	if req.TrunkID != nil && *req.TrunkID == uuid.Nil {
		return apperror.NewBadRequest("trunk_id is invalid")
	}
	if req.Destination == "" {
		return apperror.NewBadRequest("destination is required")
	}
	return nil
}

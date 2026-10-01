package conferences

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func validateOrganizationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	return nil
}

func validateCreateRequest(req *CreateRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return apperror.NewBadRequest("name is required")
	}
	return nil
}

func validateParticipantRequest(req *AddParticipantRequest) error {
	if req.CallParticipantID == nil || *req.CallParticipantID == uuid.Nil {
		return apperror.NewBadRequest("call_participant_id is required")
	}
	return nil
}

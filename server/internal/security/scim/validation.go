package scim

import (
	"net/mail"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func normalizeCreateToken(req CreateTokenRequest, now time.Time) (CreateTokenRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 128 {
		return CreateTokenRequest{}, apperror.NewBadRequest("name must be between 1 and 128 characters")
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(now) {
		return CreateTokenRequest{}, apperror.NewBadRequest("expires_at must be in the future")
	}
	return req, nil
}

func normalizeUserInput(input UserInput) (UserInput, bool, error) {
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	input.UserName = strings.ToLower(strings.TrimSpace(input.UserName))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.UserName == "" || len(input.UserName) > 320 {
		return UserInput{}, false, apperror.NewBadRequest("SCIM userName must be between 1 and 320 characters")
	}
	address, err := mail.ParseAddress(input.UserName)
	if err != nil || !strings.EqualFold(address.Address, input.UserName) {
		return UserInput{}, false, apperror.NewBadRequest("SCIM userName must be an email address")
	}
	if len(input.ExternalID) > 255 {
		return UserInput{}, false, apperror.NewBadRequest("SCIM externalId cannot exceed 255 characters")
	}
	if len(input.DisplayName) > 255 {
		return UserInput{}, false, apperror.NewBadRequest("SCIM displayName cannot exceed 255 characters")
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	return input, active, nil
}

func normalizeGroupInput(input GroupInput) (GroupInput, []uuid.UUID, error) {
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.DisplayName == "" || len(input.DisplayName) > 255 {
		return GroupInput{}, nil, apperror.NewBadRequest("SCIM group displayName must be between 1 and 255 characters")
	}
	if len(input.ExternalID) > 255 {
		return GroupInput{}, nil, apperror.NewBadRequest("SCIM externalId cannot exceed 255 characters")
	}
	memberIDs := make([]uuid.UUID, 0, len(input.Members))
	seen := make(map[uuid.UUID]struct{}, len(input.Members))
	for _, member := range input.Members {
		id, err := uuid.Parse(strings.TrimSpace(member.Value))
		if err != nil {
			return GroupInput{}, nil, apperror.NewBadRequest("SCIM group member value must be a user id")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		memberIDs = append(memberIDs, id)
	}
	return input, memberIDs, nil
}

func page(startIndex, count int) (int, int32, int32) {
	if startIndex < 1 {
		startIndex = 1
	}
	if count <= 0 {
		count = defaultPageCount
	}
	if count > maxPageCount {
		count = maxPageCount
	}
	return startIndex, int32(count), int32(startIndex - 1)
}

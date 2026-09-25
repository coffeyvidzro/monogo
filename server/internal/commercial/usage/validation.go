package usage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

var sourceTypePattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)

func validateRecord(req *RecordRequest) error {
	if req.OrganizationID == uuid.Nil || req.MeterID == uuid.Nil {
		return apperror.NewBadRequest("organization and meter are required")
	}
	if req.Quantity <= 0 {
		return apperror.NewBadRequest("usage quantity must be positive")
	}
	if !sourceTypePattern.MatchString(req.SourceType) {
		return apperror.NewBadRequest("invalid usage source type")
	}
	if strings.TrimSpace(req.SourceID) == "" {
		return apperror.NewBadRequest("usage source id is required")
	}
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 255 ||
		req.IdempotencyKey != strings.TrimSpace(req.IdempotencyKey) {
		return apperror.NewBadRequest("valid usage idempotency key is required")
	}
	if req.OccurredAt.IsZero() {
		return apperror.NewBadRequest("usage occurrence time is required")
	}
	if len(req.Dimensions) == 0 {
		req.Dimensions = json.RawMessage("{}")
	}
	if len(req.Dimensions) > 16384 {
		return apperror.NewBadRequest("usage dimensions are too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(req.Dimensions))
	var dimensions map[string]json.RawMessage
	if err := decoder.Decode(&dimensions); err != nil || dimensions == nil {
		return apperror.NewBadRequest("usage dimensions must be a JSON object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return apperror.NewBadRequest("usage dimensions must contain one JSON object")
	} else if !errors.Is(err, io.EOF) {
		return apperror.NewBadRequest("usage dimensions must contain one JSON object")
	}
	return nil
}

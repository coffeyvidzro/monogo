package providercharges

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func normalizeRecordCDRRequest(req *RecordCDRRequest) error {
	if req.ProviderID == uuid.Nil {
		return fmt.Errorf("provider id is required")
	}
	if req.CarrierConnectionID == uuid.Nil {
		return fmt.Errorf("carrier connection id is required")
	}
	if req.CallID == uuid.Nil {
		return fmt.Errorf("call id is required")
	}

	req.ProviderCDRID = strings.TrimSpace(req.ProviderCDRID)
	if len(req.ProviderCDRID) < 1 || len(req.ProviderCDRID) > 255 {
		return fmt.Errorf("provider CDR id must be between 1 and 255 characters")
	}

	direction, err := normalizeDirection(req.Direction)
	if err != nil {
		return err
	}
	req.Direction = direction

	req.Source = normalizeOptionalString(req.Source)
	req.Destination = normalizeOptionalString(req.Destination)

	if req.StartedAt.IsZero() {
		return fmt.Errorf("started time is required")
	}
	if req.EndedAt.IsZero() {
		return fmt.Errorf("ended time is required")
	}
	if req.EndedAt.Before(req.StartedAt) {
		return fmt.Errorf("ended time cannot precede started time")
	}
	if req.AnsweredAt != nil {
		if req.AnsweredAt.Before(req.StartedAt) {
			return fmt.Errorf("answered time cannot precede started time")
		}
		if req.AnsweredAt.After(req.EndedAt) {
			return fmt.Errorf("answered time cannot follow ended time")
		}
	}
	if req.DurationSeconds < 0 {
		return fmt.Errorf("duration cannot be negative")
	}
	if req.BillableSeconds < 0 {
		return fmt.Errorf("billable duration cannot be negative")
	}

	if len(req.RawPayload) == 0 {
		req.RawPayload = []byte("{}")
	}
	var payload map[string]any
	if err := json.Unmarshal(req.RawPayload, &payload); err != nil || payload == nil {
		return fmt.Errorf("raw payload must be a JSON object")
	}

	return nil
}

func normalizeRecordVoiceChargeRequest(req *RecordVoiceChargeRequest) error {
	if req.ProviderCDRID == uuid.Nil {
		return fmt.Errorf("provider CDR id is required")
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("currency must be a three-letter code")
	}
	if req.RateMicros < 0 {
		return fmt.Errorf("rate cannot be negative")
	}
	if req.BillableSeconds < 0 {
		return fmt.Errorf("billable duration cannot be negative")
	}
	if req.IncurredAt.IsZero() {
		return fmt.Errorf("incurred time is required")
	}
	if req.AmountMicros < 0 {
		return fmt.Errorf("amount cannot be negative")
	}

	return nil
}

func normalizeDirection(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case DirectionInbound, DirectionOutbound:
		return value, nil
	default:
		return "", fmt.Errorf("direction must be inbound or outbound")
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}

	return &normalized
}

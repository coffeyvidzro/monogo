package smpp

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var receiptField = regexp.MustCompile(`(?i)(?:^|\s)(id|sub|dlvrd|submit date|done date|stat|err|text):`)

// ParseDeliveryReceipt parses the SMPP v3.4 Appendix B textual receipt format.
func ParseDeliveryReceipt(raw string) (Delivery, error) {
	values := make(map[string]string)
	matches := receiptField.FindAllStringSubmatchIndex(raw, -1)
	for i, match := range matches {
		end := len(raw)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		key := strings.ToLower(raw[match[2]:match[3]])
		values[key] = strings.TrimSpace(raw[match[1]:end])
	}
	id := strings.TrimSpace(values["id"])
	state := normalizeDeliveryState(values["stat"])
	if id == "" || state == "" {
		return Delivery{}, fmt.Errorf("%w: id and stat are required", ErrMalformedReceipt)
	}
	return Delivery{MessageID: id, State: state, ErrorCode: values["err"], SubmittedAt: parseReceiptTime(values["submit date"]), DoneAt: parseReceiptTime(values["done date"]), Raw: raw}, nil
}

func normalizeDeliveryState(value string) DeliveryState {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ENROUTE":
		return DeliveryEnroute
	case "DELIVRD", "DELIVERED":
		return DeliveryDelivered
	case "EXPIRED":
		return DeliveryExpired
	case "DELETED":
		return DeliveryDeleted
	case "UNDELIV", "UNDELIVERABLE":
		return DeliveryUndeliverable
	case "ACCEPTD", "ACCEPTED":
		return DeliveryAccepted
	case "UNKNOWN":
		return DeliveryUnknown
	case "REJECTD", "REJECTED":
		return DeliveryRejected
	default:
		return ""
	}
}
func parseReceiptTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{"0601021504", "060102150405"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return &parsed
		}
	}
	return nil
}

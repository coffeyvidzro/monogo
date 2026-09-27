package smpp

import (
	"errors"
	"testing"
)

func TestParseDeliveryReceipt(t *testing.T) {
	receipt, err := ParseDeliveryReceipt("id:abc123 sub:001 dlvrd:001 submit date:2609271200 done date:2609271201 stat:DELIVRD err:000 text:hello world")
	if err != nil {
		t.Fatalf("ParseDeliveryReceipt() error = %v", err)
	}
	if receipt.MessageID != "abc123" || receipt.State != DeliveryDelivered || receipt.ErrorCode != "000" || receipt.SubmittedAt == nil || receipt.DoneAt == nil {
		t.Fatalf("receipt = %#v", receipt)
	}
	if receipt.SubmittedAt.UTC().Format("2006-01-02 15:04") != "2026-09-27 12:00" {
		t.Fatalf("submitted_at = %s", receipt.SubmittedAt)
	}
}

func TestParseDeliveryReceiptRejectsMalformed(t *testing.T) {
	_, err := ParseDeliveryReceipt("not a receipt")
	if !errors.Is(err, ErrMalformedReceipt) {
		t.Fatalf("error = %v", err)
	}
}

func TestNormalizeDeliveryStates(t *testing.T) {
	for input, want := range map[string]DeliveryState{"ENROUTE": DeliveryEnroute, "DELIVRD": DeliveryDelivered, "EXPIRED": DeliveryExpired, "DELETED": DeliveryDeleted, "UNDELIV": DeliveryUndeliverable, "ACCEPTD": DeliveryAccepted, "UNKNOWN": DeliveryUnknown, "REJECTD": DeliveryRejected} {
		if got := normalizeDeliveryState(input); got != want {
			t.Errorf("normalizeDeliveryState(%q) = %q, want %q", input, got, want)
		}
	}
	if parseReceiptTime("bad") != nil {
		t.Fatal("invalid receipt time accepted")
	}
}

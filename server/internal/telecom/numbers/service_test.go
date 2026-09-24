package numbers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestManagedNumberResponseHidesPlatformCarrier(t *testing.T) {
	connectionID := uuid.New()
	encoded, err := json.Marshal(response(sqlc.PhoneNumber{
		ID:                  uuid.New(),
		OrganizationID:      uuid.New(),
		Number:              "+15551234567",
		CountryCode:         "US",
		ProvisioningMode:    "managed",
		CarrierConnectionID: &connectionID,
		Status:              "active",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if body["type"] != "managed" {
		t.Fatalf("managed number type = %v", body["type"])
	}
	if _, exposed := body["carrier_connection_id"]; exposed {
		t.Fatal("managed response exposed the platform carrier connection")
	}
	if _, exposed := body["provisioning_mode"]; exposed {
		t.Fatal("managed response exposed internal provisioning terminology")
	}
}

func TestNormalizeBYOC(t *testing.T) {
	request := CreateBYOCRequest{
		Number:      " +233201234567 ",
		CountryCode: " gh ",
	}
	if err := normalizeBYOC(&request); err != nil {
		t.Fatal(err)
	}
	if request.Number != "+233201234567" || request.CountryCode != "GH" {
		t.Fatalf("unexpected normalized request: %+v", request)
	}
}

func TestNormalizeBYOCRejectsInvalidValues(t *testing.T) {
	tests := []CreateBYOCRequest{
		{Number: "0201234567", CountryCode: "GH"},
		{Number: "+023201234567", CountryCode: "GH"},
		{Number: "+233201234567", CountryCode: "GHA"},
		{Number: "+233201234567", CountryCode: "GH", CarrierConnectionID: new(uuid.UUID)},
	}
	for _, req := range tests {
		if err := normalizeBYOC(&req); err == nil {
			t.Fatalf("expected validation error for %+v", req)
		}
	}
}

func TestValidateUpdateRequiresCapability(t *testing.T) {
	if err := validateUpdate(UpdateRequest{}); err == nil {
		t.Fatal("expected missing capability error")
	}
	disabled := false
	if err := validateUpdate(UpdateRequest{VoiceEnabled: &disabled}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateIDsRejectsMissingIdentifiers(t *testing.T) {
	if err := validateIDs(uuid.Nil, uuid.New()); err == nil {
		t.Fatal("expected missing organization error")
	}
	if err := validateIDs(uuid.New(), uuid.Nil); err == nil {
		t.Fatal("expected missing number error")
	}
}

func TestNumberErrors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{pgx.ErrNoRows, "NOT_FOUND"},
		{&pgconn.PgError{Code: "23505"}, "CONFLICT"},
		{&pgconn.PgError{Code: "23514"}, "BAD_REQUEST"},
	} {
		var appErr *apperror.AppError
		if err := writeError(tc.err); !errors.As(err, &appErr) || appErr.Code != tc.code {
			t.Fatalf("writeError(%v) = %v, want %s", tc.err, err, tc.code)
		}
	}
}

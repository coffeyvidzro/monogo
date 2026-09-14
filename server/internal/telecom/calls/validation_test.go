package calls

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateCreateRequest(t *testing.T) {
	t.Parallel()

	trunkID := uuid.New()
	tests := []struct {
		name string
		req  CreateCallRequest
		want bool
	}{
		{name: "valid BYOC request", req: CreateCallRequest{TrunkID: ptr(trunkID), From: "1000", To: "1001"}, want: true},
		{name: "valid managed request", req: CreateCallRequest{From: "1000", To: "1001"}, want: true},
		{name: "missing originator", req: CreateCallRequest{TrunkID: ptr(trunkID), To: "1001"}},
		{name: "nil trunk rejected", req: CreateCallRequest{TrunkID: ptr(uuid.Nil), From: "1000", To: "1001"}},
		{name: "nil application rejected", req: CreateCallRequest{ApplicationID: ptr(uuid.Nil), TrunkID: ptr(trunkID), From: "1000", To: "1001"}},
		{name: "invalid DTMF mode", req: CreateCallRequest{TrunkID: ptr(trunkID), From: "1000", To: "1001", DTMFMode: "inband"}},
		{name: "unsupported codec", req: CreateCallRequest{TrunkID: ptr(trunkID), From: "1000", To: "1001", Codecs: []string{"G729"}}},
		{name: "duplicate codec", req: CreateCallRequest{TrunkID: ptr(trunkID), From: "1000", To: "1001", Codecs: []string{"PCMU", "pcmu"}}},
		{name: "invalid media encryption", req: CreateCallRequest{TrunkID: ptr(trunkID), From: "1000", To: "1001", MediaEncryption: "dtls"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateCreateRequest(&tt.req); (err == nil) != tt.want {
				t.Fatalf("validateCreateRequest() error = %v, want success %t", err, tt.want)
			}
		})
	}
	request := CreateCallRequest{TrunkID: ptr(trunkID), From: "1000", To: "1001"}
	if err := validateCreateRequest(&request); err != nil {
		t.Fatalf("validate defaults: %v", err)
	}
	if request.DTMFMode != "rfc2833" || request.MediaEncryption != "rtp" || len(request.Codecs) != 4 {
		t.Fatalf("unexpected media defaults: %+v", request)
	}
}

func TestValidateRecordRequest(t *testing.T) {
	t.Parallel()

	request := RecordRequest{Path: "/recordings/call.wav"}
	if err := validateRecordRequest(&request); err != nil {
		t.Fatalf("validateRecordRequest() error = %v", err)
	}
	if request.Action != "start" {
		t.Fatalf("action = %q, want start", request.Action)
	}
	if err := validateRecordRequest(&RecordRequest{Action: "stop"}); err != nil {
		t.Fatalf("stop recording should not require a path: %v", err)
	}
}

func ptr(value uuid.UUID) *uuid.UUID { return &value }

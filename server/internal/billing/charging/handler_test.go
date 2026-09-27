package charging

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDecodeRequest(t *testing.T) {
	t.Parallel()

	operationID := uuid.New()
	request := requestWithBody(
		t,
		`{"operation_id":"`+operationID.String()+`","amount_micros":2500}`,
	)

	var body OperationRequest
	if err := decodeRequest(request, &body); err != nil {
		t.Fatalf(
			"decode valid request: %v",
			err,
		)
	}
	if body.OperationID != operationID {
		t.Fatalf(
			"operation id = %s, want %s",
			body.OperationID,
			operationID,
		)
	}
	if body.AmountMicros != 2_500 {
		t.Fatalf(
			"amount = %d, want 2500",
			body.AmountMicros,
		)
	}
}

func TestDecodeRequestRejectsUnknownAndTrailingValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "unknown field",
			body: `{"operation_id":"` + uuid.NewString() + `","amount_micros":1,"amount":1}`,
		},
		{
			name: "trailing value",
			body: `{"operation_id":"` + uuid.NewString() + `","amount_micros":1} {}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := requestWithBody(t, test.body)
			if err := decodeRequest(request, &OperationRequest{}); err == nil {
				t.Fatal("expected invalid request body")
			}
		})
	}
}

func TestDefaultOccurredAt(t *testing.T) {
	t.Parallel()

	var occurredAt time.Time
	defaultOccurredAt(&occurredAt)
	if occurredAt.IsZero() {
		t.Fatal("expected occurrence time to be defaulted")
	}

	explicit := time.Date(
		2026,
		time.September,
		27,
		12,
		0,
		0,
		0,
		time.UTC,
	)
	defaultOccurredAt(&explicit)
	if !explicit.Equal(time.Date(
		2026,
		time.September,
		27,
		12,
		0,
		0,
		0,
		time.UTC,
	)) {
		t.Fatal("expected explicit occurrence time to be preserved")
	}
}

func requestWithBody(t *testing.T, body string) *http.Request {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf(
			"create request: %v",
			err,
		)
	}

	return request
}

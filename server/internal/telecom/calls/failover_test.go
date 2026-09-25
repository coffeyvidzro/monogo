package calls

import (
	"context"
	"errors"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
)

func TestSIPFailoverAcceptancePrimarySecondaryTertiary(t *testing.T) {
	routes := testRoutes(3)
	responses := []error{
		&calling.OriginateError{Class: calling.OriginateFailureTransport, Err: errors.New("TCP reset")},
		calling.NewSIPOriginateError(503, errors.New("carrier unavailable")),
		nil,
	}
	var outcomes []routeAttemptOutcome
	result, selected, err := executeRoutePlan(context.Background(), routes, func(
		_ context.Context,
		route routing.OutboundRoute,
	) (calling.OriginateResult, error) {
		index := len(outcomes)
		if responses[index] != nil {
			return calling.OriginateResult{}, responses[index]
		}
		return calling.OriginateResult{ChannelID: "channel-tertiary"}, nil
	}, func(_ context.Context, outcome routeAttemptOutcome) {
		outcomes = append(outcomes, outcome)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChannelID != "channel-tertiary" || selected.TrunkEndpointID != routes[2].TrunkEndpointID {
		t.Fatalf("unexpected selected route: result=%+v route=%+v", result, selected)
	}
	if len(outcomes) != 3 || outcomes[0].Outcome != "retryable_failure" ||
		outcomes[1].SIPStatus != 503 || outcomes[2].Outcome != "succeeded" {
		t.Fatalf("unexpected attempt outcomes: %+v", outcomes)
	}
}

func TestExecuteRoutePlanStopsOnTerminalSIPResponse(t *testing.T) {
	routes := testRoutes(3)
	attempts := 0
	_, _, err := executeRoutePlan(context.Background(), routes, func(
		_ context.Context,
		_ routing.OutboundRoute,
	) (calling.OriginateResult, error) {
		attempts++
		return calling.OriginateResult{}, calling.NewSIPOriginateError(403, errors.New("forbidden"))
	}, nil)
	if err == nil || attempts != 1 {
		t.Fatalf("terminal response made %d attempts, err=%v", attempts, err)
	}
}

func TestExecuteRoutePlanCapsAttemptBudgetAtThree(t *testing.T) {
	routes := testRoutes(5)
	attempts := 0
	_, _, err := executeRoutePlan(context.Background(), routes, func(
		_ context.Context,
		_ routing.OutboundRoute,
	) (calling.OriginateResult, error) {
		attempts++
		return calling.OriginateResult{}, &calling.OriginateError{
			Class: calling.OriginateFailureTimeout, Err: context.DeadlineExceeded,
		}
	}, nil)
	if err == nil || attempts != maxOutboundAttempts {
		t.Fatalf("attempt budget used %d routes, err=%v", attempts, err)
	}
}

func TestClassifyOriginateFailure(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{name: "timeout", err: &calling.OriginateError{Class: calling.OriginateFailureTimeout, Err: context.DeadlineExceeded}, retryable: true},
		{name: "capacity", err: &calling.OriginateError{Class: calling.OriginateFailureCapacity, Err: errors.New("CPS")}, retryable: true},
		{name: "server response", err: calling.NewSIPOriginateError(500, errors.New("server error")), retryable: true},
		{name: "client response", err: calling.NewSIPOriginateError(486, errors.New("busy")), retryable: false},
		{name: "validation", err: &calling.OriginateError{Class: calling.OriginateFailureValidation, Err: errors.New("bad route")}, retryable: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, outcome := classifyOriginateFailure(test.err)
			if got := outcome == "retryable_failure"; got != test.retryable {
				t.Fatalf("retryable=%v, want %v", got, test.retryable)
			}
		})
	}
}

func testRoutes(count int) []routing.OutboundRoute {
	routes := make([]routing.OutboundRoute, count)
	for index := range routes {
		routes[index] = routing.OutboundRoute{
			CarrierConnectionID: uuid.New(), TrunkID: uuid.New(), TrunkEndpointID: uuid.New(),
		}
	}
	return routes
}

package calls

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
)

const maxOutboundAttempts = 3

type routeAttemptOutcome struct {
	Route        routing.OutboundRoute
	Attempt      int
	Outcome      string
	FailureClass string
	SIPStatus    int
	Duration     time.Duration
	Err          error
}

type routeAttemptFunc func(context.Context, routing.OutboundRoute) (calling.OriginateResult, error)
type routeAttemptObserver func(context.Context, routeAttemptOutcome)

func executeRoutePlan(
	ctx context.Context,
	routes []routing.OutboundRoute,
	attempt routeAttemptFunc,
	observe routeAttemptObserver,
) (calling.OriginateResult, routing.OutboundRoute, error) {
	limit := min(len(routes), maxOutboundAttempts)
	var lastErr error
	for index := 0; index < limit; index++ {
		if err := ctx.Err(); err != nil {
			return calling.OriginateResult{}, routing.OutboundRoute{}, err
		}
		route := routes[index]
		started := time.Now()
		result, err := attempt(ctx, route)
		outcome := routeAttemptOutcome{
			Route: route, Attempt: index + 1, Duration: time.Since(started), Err: err,
		}
		if err == nil {
			outcome.Outcome = "succeeded"
			if observe != nil {
				observe(context.WithoutCancel(ctx), outcome)
			}
			return result, route, nil
		}

		outcome.FailureClass, outcome.SIPStatus, outcome.Outcome = classifyOriginateFailure(err)
		if observe != nil {
			observe(context.WithoutCancel(ctx), outcome)
		}
		lastErr = err
		if outcome.Outcome != "retryable_failure" {
			break
		}
	}
	if lastErr == nil {
		lastErr = errors.New("outbound route plan is empty")
	}
	return calling.OriginateResult{}, routing.OutboundRoute{}, lastErr
}

func classifyOriginateFailure(err error) (failureClass string, sipStatus int, outcome string) {
	outcome = "terminal_failure"
	var originateError *calling.OriginateError
	if !errors.As(err, &originateError) {
		return string(calling.OriginateFailureInternal), 0, outcome
	}
	failureClass = string(originateError.Class)
	sipStatus = originateError.SIPStatus
	if originateError.Class == calling.OriginateFailureTransport ||
		originateError.Class == calling.OriginateFailureTimeout ||
		originateError.Class == calling.OriginateFailureCapacity ||
		(originateError.Class == calling.OriginateFailureSIP && sipStatus >= 500 && sipStatus <= 599) {
		outcome = "retryable_failure"
	}
	return failureClass, sipStatus, outcome
}

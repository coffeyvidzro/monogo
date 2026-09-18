package worker

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
)

var freeSWITCHLifecycleEvents = []string{
	"CHANNEL_CREATE",
	"CHANNEL_ANSWER",
	"CHANNEL_HOLD",
	"CHANNEL_UNHOLD",
	"CHANNEL_HANGUP_COMPLETE",
	"RECORD_START",
	"RECORD_STOP",
}

func subscribeFreeSWITCH(ctx context.Context, logger *logging.Logger, modules *modules) error {
	return modules.freeSwitch.Subscribe(
		ctx,
		freeswitch.EventFormatPlain,
		freeSWITCHLifecycleEvents,
		func(eventCtx context.Context, event freeswitch.Event) error {
			admittedInbound := false
			if event.Name == "CHANNEL_CREATE" &&
				event.Header("variable_leamout_call_id") == "" {
				admission, err := calls.TranslateInboundFreeSWITCHEvent(event)
				switch {
				case errors.Is(err, calls.ErrNotInboundAdmission):
				case err != nil:
					logger.Error(
						eventCtx,
						"translate inbound FreeSWITCH event",
						"error", err,
					)
					if channelID := event.Header("Unique-ID"); channelID != "" {
						if hangupErr := modules.freeSwitch.Hangup(eventCtx, channelID); hangupErr != nil {
							logger.Error(
								eventCtx,
								"reject malformed inbound FreeSWITCH channel",
								"channel_id", channelID,
								"error", hangupErr,
							)
						}
					}
				default:
					if _, err := modules.callsService.AdmitInbound(eventCtx, admission); err != nil {
						logger.Error(
							eventCtx,
							"admit inbound call",
							"channel_id", admission.ChannelID,
							"error", err,
						)
					}
					admittedInbound = true
				}
			}

			if !admittedInbound {
				callEvent, err := calls.TranslateFreeSWITCHEvent(event)
				switch {
				case errors.Is(err, calls.ErrUnsupportedEvent),
					errors.Is(err, calls.ErrUncorrelatedEvent):
				case err != nil:
					logger.Error(
						eventCtx,
						"translate call FreeSWITCH event",
						"event", event.Name,
						"error", err,
					)
				default:
					if err := modules.callsService.ObserveLifecycle(eventCtx, callEvent); err != nil {
						logger.Error(
							eventCtx,
							"handle call FreeSWITCH event",
							"event", event.Name,
							"error", err,
						)
					}
				}
			}

			if err := modules.recordingConsumer.HandleFreeSWITCHEvent(eventCtx, event); err != nil {
				logger.Error(eventCtx, "handle recording FreeSWITCH event", "event", event.Name, "error", err)
			}
			return nil
		},
	)
}

func runWorkloads(ctx context.Context, logger *logging.Logger, modules *modules) error {
	group, groupCtx := errgroup.WithContext(ctx)

	run := func(name string, job func(context.Context) error) {
		group.Go(func() error {
			logger.Info(groupCtx, name+" started")
			if err := job(groupCtx); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			return nil
		})
	}

	run("outbox publisher", modules.outbox.Run)
	run("webhook consumer", modules.webhookConsumer.Run)
	run("webhook delivery worker", modules.webhookDelivery.Run)
	run("call reconciliation", modules.callReconciliation.Run)
	run("recording reconciliation", modules.recordingReconciliation.Run)
	run("idempotency cleanup", modules.idempotencyCleanup.Run)

	return group.Wait()
}

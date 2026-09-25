package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
)

var freeSWITCHLifecycleEvents = append(
	calling.FreeSWITCHEvents(),
	"RECORD_START",
	"RECORD_STOP",
)

func subscribeFreeSWITCH(
	ctx context.Context,
	logger *logging.Logger,
	modules *modules,
) error {
	return modules.freeSwitch.Subscribe(
		ctx,
		freeswitch.EventFormatPlain,
		freeSWITCHLifecycleEvents,
		func(eventCtx context.Context, event freeswitch.Event) error {
			if event.Name == "CHANNEL_CREATE" &&
				strings.TrimSpace(event.Header("variable_leamout_call_id")) == "" {
				admission, err := calling.TranslateInboundFreeSWITCHEvent(event)
				switch {
				case errors.Is(err, calling.ErrNotInboundAdmission):
				case err != nil:
					logger.Error(
						eventCtx,
						"translate inbound FreeSWITCH event",
						"error", err,
					)
					if channelID := strings.TrimSpace(event.Header("Unique-ID")); channelID != "" {
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
					if err := modules.callConsumer.HandleInbound(eventCtx, admission); err != nil {
						logger.Error(
							eventCtx,
							"handle inbound call event",
							"channel_id", admission.ChannelID,
							"error", err,
						)
					}
					return nil
				}
			}

			callEvent, err := calling.TranslateFreeSWITCHEvent(event)
			switch {
			case errors.Is(err, calling.ErrUnsupportedEvent),
				errors.Is(err, calling.ErrUncorrelatedEvent):
			case err != nil:
				logger.Error(
					eventCtx,
					"translate call FreeSWITCH event",
					"event", event.Name,
					"error", err,
				)
			default:
				if err := modules.callConsumer.HandleLifecycle(eventCtx, callEvent); err != nil {
					logger.Error(
						eventCtx,
						"handle call lifecycle event",
						"event", event.Name,
						"error", err,
					)
				}
			}

			if err := modules.recordingConsumer.HandleFreeSWITCHEvent(eventCtx, event); err != nil {
				logger.Error(
					eventCtx,
					"handle recording FreeSWITCH event",
					"event", event.Name,
					"error", err,
				)
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
	run("recording ingestion", modules.recordingIngestion.Run)
	run("idempotency cleanup", modules.idempotencyCleanup.Run)
	run("trunk endpoint health checks", modules.trunkHealth.Run)
	if modules.numberReconciliation != nil {
		run("managed number reconciliation", modules.numberReconciliation.Run)
	}
	if modules.lifecycleReconciliation != nil {
		run("number lifecycle reconciliation", modules.lifecycleReconciliation.Run)
	}

	return group.Wait()
}

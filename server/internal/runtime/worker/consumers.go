package worker

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
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
			if err := modules.callConsumer.HandleFreeSWITCHEvent(eventCtx, event); err != nil {
				logger.Error(eventCtx, "handle call FreeSWITCH event", "event", event.Name, "error", err)
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
	// run("call reconciliation", modules.callReconciliation.Run)
	run("recording reconciliation", modules.recordingReconciliation.Run)
	run("idempotency cleanup", modules.idempotencyCleanup.Run)

	return group.Wait()
}

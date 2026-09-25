package lifecycle

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/google/uuid"
)

func insertNumberEvent(ctx context.Context, queries *sqlc.Queries, subject string, organizationID, aggregateID uuid.UUID, resource any) error {
	payload := map[string]any{"event_type": subject, "organization_id": organizationID, "resource": resource, "occurred_at": time.Now().UTC()}
	_, err := outbox.NewRepository(queries).Insert(ctx, outbox.Event{
		Subject: subject, AggregateType: "number_lifecycle", AggregateID: aggregateID,
		Payload: payload, Headers: map[string]string{"event_type": subject, "organization_id": organizationID.String()},
	})
	return err
}

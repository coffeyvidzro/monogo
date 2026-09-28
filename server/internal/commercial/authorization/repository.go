package authorization

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	if db == nil {
		panic("commercial authorization: database is required")
	}

	return &Repository{
		db: db,
	}
}

func (r *Repository) PersistFreeCallAuthorization(
	ctx context.Context,
	callID uuid.UUID,
	organizationID uuid.UUID,
	rate pricing.Rate,
	authorizedAt time.Time,
) (CallAuthorization, error) {
	row := r.db.QueryRow(
		ctx,
		`UPDATE calls
SET
    customer_carrier_rate_id = $3,
    customer_rate_currency = $4,
    customer_rate_micros = $5,
    customer_rate_billing_unit = $6,
    customer_rate_billing_increment_seconds = $7,
    customer_rate_minimum_duration_seconds = $8,
    commercial_authorized_at = $9,
    updated_at = NOW()
WHERE id = $1
  AND organization_id = $2
  AND direction = 'outbound'
  AND state = 'initiating'
  AND (
      commercial_authorized_at IS NULL
      OR (
          customer_carrier_rate_id = $3
          AND customer_rate_currency = $4
          AND customer_rate_micros = $5
          AND customer_rate_billing_unit = $6
          AND customer_rate_billing_increment_seconds = $7
          AND customer_rate_minimum_duration_seconds = $8
      )
  )
RETURNING
    id,
    organization_id,
    customer_carrier_rate_id,
    customer_rate_currency,
    customer_rate_micros,
    customer_rate_billing_unit,
    customer_rate_billing_increment_seconds,
    customer_rate_minimum_duration_seconds,
    commercial_authorized_at`,
		callID,
		organizationID,
		rate.ID,
		rate.Currency,
		rate.RateMicros,
		rate.BillingUnit,
		rate.BillingIncrementSeconds,
		rate.MinimumDurationSeconds,
		authorizedAt,
	)

	var result CallAuthorization
	var authorizedAtValue pgtype.Timestamptz
	err := row.Scan(
		&result.CallID,
		&result.OrganizationID,
		&result.CarrierRateID,
		&result.Currency,
		&result.RateMicros,
		&result.BillingUnit,
		&result.BillingIncrementSeconds,
		&result.MinimumDurationSeconds,
		&authorizedAtValue,
	)
	if err != nil {
		return CallAuthorization{}, err
	}
	result.AuthorizedAt = pgconv.TimestamptzToTime(authorizedAtValue)

	return result, nil
}

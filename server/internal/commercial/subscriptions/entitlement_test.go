package subscriptions

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequireActiveAllowsCurrentActiveSubscription(t *testing.T) {
	now := time.Date(
		2026,
		time.September,
		28,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	service, organizationID := newEntitlementTestService(
		t,
		entitlementFixture{
			Status:             StatusActive,
			CurrentPeriodStart: now.Add(-time.Hour),
			CurrentPeriodEnd:   now.Add(time.Hour),
		},
	)
	service.now = func() time.Time {
		return now
	}

	if err := service.RequireActive(
		context.Background(),
		organizationID,
	); err != nil {
		t.Fatalf("require active subscription: %v", err)
	}
}

func TestRequireActiveAllowsCancelAtPeriodEndUntilPeriodEnds(t *testing.T) {
	now := time.Date(
		2026,
		time.September,
		28,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	service, organizationID := newEntitlementTestService(
		t,
		entitlementFixture{
			Status:             StatusActive,
			CurrentPeriodStart: now.Add(-time.Hour),
			CurrentPeriodEnd:   now.Add(time.Hour),
			CancelAtPeriodEnd:  true,
		},
	)
	service.now = func() time.Time {
		return now
	}

	if err := service.RequireActive(
		context.Background(),
		organizationID,
	); err != nil {
		t.Fatalf("require active subscription: %v", err)
	}
}

func TestRequireActiveRejectsNonActiveSubscriptionStates(t *testing.T) {
	now := time.Date(
		2026,
		time.September,
		28,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	for _, status := range []string{
		StatusPending,
		StatusPastDue,
		StatusCancelled,
	} {
		t.Run(status, func(t *testing.T) {
			service, organizationID := newEntitlementTestService(
				t,
				entitlementFixture{
					Status:             status,
					CurrentPeriodStart: now.Add(-time.Hour),
					CurrentPeriodEnd:   now.Add(time.Hour),
				},
			)
			service.now = func() time.Time {
				return now
			}

			err := service.RequireActive(
				context.Background(),
				organizationID,
			)
			requireSubscriptionErrorCode(
				t,
				err,
				"PAYMENT_REQUIRED",
			)
		})
	}
}

func TestRequireActiveRejectsExpiredSubscription(t *testing.T) {
	now := time.Date(
		2026,
		time.September,
		28,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	service, organizationID := newEntitlementTestService(
		t,
		entitlementFixture{
			Status:             StatusActive,
			CurrentPeriodStart: now.Add(-2 * time.Hour),
			CurrentPeriodEnd:   now,
		},
	)
	service.now = func() time.Time {
		return now
	}

	err := service.RequireActive(
		context.Background(),
		organizationID,
	)
	requireSubscriptionErrorCode(
		t,
		err,
		"PAYMENT_REQUIRED",
	)
}

func TestRequireActiveRejectsFutureSubscription(t *testing.T) {
	now := time.Date(
		2026,
		time.September,
		28,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	service, organizationID := newEntitlementTestService(
		t,
		entitlementFixture{
			Status:             StatusActive,
			CurrentPeriodStart: now.Add(time.Minute),
			CurrentPeriodEnd:   now.Add(time.Hour),
		},
	)
	service.now = func() time.Time {
		return now
	}

	err := service.RequireActive(
		context.Background(),
		organizationID,
	)
	requireSubscriptionErrorCode(
		t,
		err,
		"PAYMENT_REQUIRED",
	)
}

func TestRequireActiveRejectsMissingSubscription(t *testing.T) {
	now := time.Date(
		2026,
		time.September,
		28,
		20,
		0,
		0,
		0,
		time.UTC,
	)

	service, organizationID := newEntitlementTestService(
		t,
		entitlementFixture{},
	)
	service.now = func() time.Time {
		return now
	}

	err := service.RequireActive(
		context.Background(),
		organizationID,
	)
	requireSubscriptionErrorCode(
		t,
		err,
		"PAYMENT_REQUIRED",
	)
}

func TestRequireActiveRejectsMissingOrganizationID(t *testing.T) {
	service := &Service{}

	err := service.RequireActive(
		context.Background(),
		uuid.Nil,
	)
	requireSubscriptionErrorCode(
		t,
		err,
		"BAD_REQUEST",
	)
}

type entitlementFixture struct {
	Status             string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
}

func newEntitlementTestService(
	t *testing.T,
	fixture entitlementFixture,
) (*Service, uuid.UUID) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(
		ctx,
		databaseURL,
	)
	if err != nil {
		t.Fatalf("open test postgres: %v", err)
	}

	schema := "subscription_test_" + uuid.New().String()
	schema = stripHyphens(schema)
	quotedSchema := pgx.Identifier{schema}.Sanitize()

	if _, err := admin.Exec(
		ctx,
		"CREATE SCHEMA "+quotedSchema,
	); err != nil {
		admin.Close()
		t.Fatalf("create test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = admin.Exec(
			ctx,
			"DROP SCHEMA "+quotedSchema+" CASCADE",
		)
		admin.Close()
		t.Fatalf("parse test database config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(
		ctx,
		config,
	)
	if err != nil {
		_, _ = admin.Exec(
			ctx,
			"DROP SCHEMA "+quotedSchema+" CASCADE",
		)
		admin.Close()
		t.Fatalf("open isolated test pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(
			context.Background(),
			"DROP SCHEMA "+quotedSchema+" CASCADE",
		)
		admin.Close()
	})

	createEntitlementTestSchema(
		t,
		pool,
	)

	organizationID := uuid.New()
	if _, err := pool.Exec(
		ctx,
		`INSERT INTO organizations (
			id,
			status
		) VALUES ($1, 'active')`,
		organizationID,
	); err != nil {
		t.Fatalf("insert test organization: %v", err)
	}

	if fixture.Status != "" {
		if _, err := pool.Exec(
			ctx,
			`INSERT INTO subscriptions (
				id,
				organization_id,
				plan_id,
				status,
				currency,
				amount_micros,
				interval,
				current_period_start,
				current_period_end,
				cancel_at_period_end,
				started_at
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				'USD',
				1000000,
				'month',
				$5,
				$6,
				$7,
				$5
			)`,
			uuid.New(),
			organizationID,
			uuid.New(),
			fixture.Status,
			fixture.CurrentPeriodStart,
			fixture.CurrentPeriodEnd,
			fixture.CancelAtPeriodEnd,
		); err != nil {
			t.Fatalf("insert test subscription: %v", err)
		}
	}

	queries := sqlc.New(pool)
	repository := NewRepository(queries)
	service := NewService(repository)

	return service, organizationID
}

func createEntitlementTestSchema(
	t *testing.T,
	pool *pgxpool.Pool,
) {
	t.Helper()

	statements := []string{
		`CREATE TABLE organizations (
			id UUID PRIMARY KEY,
			status TEXT NOT NULL,
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE TABLE subscriptions (
			id UUID PRIMARY KEY,
			organization_id UUID NOT NULL,
			plan_id UUID NOT NULL,
			status TEXT NOT NULL,
			currency CHAR(3) NOT NULL,
			amount_micros BIGINT NOT NULL,
			interval TEXT NOT NULL,
			current_period_start TIMESTAMPTZ,
			current_period_end TIMESTAMPTZ,
			cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
			started_at TIMESTAMPTZ,
			cancelled_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}

	for _, statement := range statements {
		if _, err := pool.Exec(
			context.Background(),
			statement,
		); err != nil {
			t.Fatalf("create subscription test schema: %v", err)
		}
	}
}

func requireSubscriptionErrorCode(
	t *testing.T,
	err error,
	want string,
) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected application error %q", want)
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %T, want *apperror.AppError", err)
	}
	if appErr.Code != want {
		t.Fatalf(
			"application error code = %q, want %q",
			appErr.Code,
			want,
		)
	}
}

func stripHyphens(value string) string {
	result := make([]byte, 0, len(value))
	for index := range len(value) {
		if value[index] != '-' {
			result = append(
				result,
				value[index],
			)
		}
	}

	return string(result)
}

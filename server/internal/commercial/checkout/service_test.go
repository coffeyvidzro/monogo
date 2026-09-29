package checkout

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSubscriptionAmountMicros int64 = 29_000_000

func TestCreateSubscriptionCheckoutSnapshotsSubscription(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		return now
	}

	result, err := service.CreateSubscription(
		context.Background(),
		CreateSubscriptionRequest{
			OrganizationID: organizationID,
			SubscriptionID: subscriptionID,
			Provider:       "stripe",
		},
	)
	if err != nil {
		t.Fatalf("create subscription checkout: %v", err)
	}

	payment := result.Payment
	if payment.Purpose != payments.PurposeSubscription {
		t.Fatalf("payment purpose = %q, want %q", payment.Purpose, payments.PurposeSubscription)
	}
	if payment.AmountMicros != testSubscriptionAmountMicros {
		t.Fatalf("payment amount = %d, want %d", payment.AmountMicros, testSubscriptionAmountMicros)
	}
	if payment.Currency != "USD" {
		t.Fatalf("payment currency = %q, want USD", payment.Currency)
	}
	if payment.SubscriptionID == nil || *payment.SubscriptionID != subscriptionID {
		t.Fatalf("payment subscription id = %v, want %s", payment.SubscriptionID, subscriptionID)
	}
	if payment.PeriodStart == nil || !payment.PeriodStart.Equal(now) {
		t.Fatalf("payment period start = %v, want %v", payment.PeriodStart, now)
	}
	wantEnd := now.AddDate(0, 1, 0)
	if payment.PeriodEnd == nil || !payment.PeriodEnd.Equal(wantEnd) {
		t.Fatalf("payment period end = %v, want %v", payment.PeriodEnd, wantEnd)
	}

	assertSubscriptionStatus(t, pool, subscriptionID, subscriptions.StatusPending)
}

func TestCompleteSubscriptionCheckoutActivatesAndReplays(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		return now
	}

	result, err := service.CreateSubscription(
		context.Background(),
		CreateSubscriptionRequest{
			OrganizationID: organizationID,
			SubscriptionID: subscriptionID,
			Provider:       "stripe",
		},
	)
	if err != nil {
		t.Fatalf("create subscription checkout: %v", err)
	}

	complete := CompleteRequest{
		OrganizationID:  organizationID,
		PaymentID:       result.Payment.ID,
		ProviderEventID: "evt_subscription_paid",
		OccurredAt:      now.Add(time.Minute),
	}
	completed, err := service.Complete(context.Background(), complete)
	if err != nil {
		t.Fatalf("complete subscription checkout: %v", err)
	}
	if completed.Payment.Status != payments.StatusSucceeded {
		t.Fatalf("payment status = %q, want %q", completed.Payment.Status, payments.StatusSucceeded)
	}

	replayed, err := service.Complete(context.Background(), complete)
	if err != nil {
		t.Fatalf("replay subscription checkout completion: %v", err)
	}
	if replayed.Payment.ID != completed.Payment.ID {
		t.Fatalf("replayed payment id = %s, want %s", replayed.Payment.ID, completed.Payment.ID)
	}

	var (
		status      string
		periodStart time.Time
		periodEnd   time.Time
	)
	if err := pool.QueryRow(
		context.Background(),
		`SELECT status, current_period_start, current_period_end
		 FROM subscriptions
		 WHERE id = $1`,
		subscriptionID,
	).Scan(&status, &periodStart, &periodEnd); err != nil {
		t.Fatalf("read activated subscription: %v", err)
	}
	if status != subscriptions.StatusActive {
		t.Fatalf("subscription status = %q, want %q", status, subscriptions.StatusActive)
	}
	if !periodStart.Equal(now) {
		t.Fatalf("subscription period start = %v, want %v", periodStart, now)
	}
	if !periodEnd.Equal(now.AddDate(0, 1, 0)) {
		t.Fatalf("subscription period end = %v, want %v", periodEnd, now.AddDate(0, 1, 0))
	}
}

func TestCompleteWalletTopupCreditsOnceAndReplays(t *testing.T) {
	service, pool, organizationID, _ := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		return now
	}

	result, err := service.CreateWalletTopup(
		context.Background(),
		CreateWalletTopupRequest{
			OrganizationID: organizationID,
			Provider:       "paystack",
			AmountMicros:   50_000_000,
		},
	)
	if err != nil {
		t.Fatalf("create wallet top-up checkout: %v", err)
	}

	complete := CompleteRequest{
		OrganizationID:  organizationID,
		PaymentID:       result.Payment.ID,
		ProviderEventID: "evt_wallet_topup",
		OccurredAt:      now.Add(time.Minute),
	}
	if _, err := service.Complete(context.Background(), complete); err != nil {
		t.Fatalf("complete wallet top-up checkout: %v", err)
	}
	if _, err := service.Complete(context.Background(), complete); err != nil {
		t.Fatalf("replay wallet top-up completion: %v", err)
	}

	var balance int64
	if err := pool.QueryRow(
		context.Background(),
		`SELECT balance_micros
		 FROM wallets
		 WHERE organization_id = $1`,
		organizationID,
	).Scan(&balance); err != nil {
		t.Fatalf("read wallet balance: %v", err)
	}
	if balance != 50_000_000 {
		t.Fatalf("wallet balance = %d, want %d", balance, int64(50_000_000))
	}

	var ledgerCount int
	if err := pool.QueryRow(
		context.Background(),
		`SELECT count(*)
		 FROM wallet_ledger_entries
		 WHERE operation_id = $1`,
		result.Payment.ID,
	).Scan(&ledgerCount); err != nil {
		t.Fatalf("count top-up ledger entries: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("top-up ledger entries = %d, want 1", ledgerCount)
	}
}

func TestFailedCheckoutDoesNotCompleteSubscription(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		return now
	}

	result, err := service.CreateSubscription(
		context.Background(),
		CreateSubscriptionRequest{
			OrganizationID: organizationID,
			SubscriptionID: subscriptionID,
			Provider:       "stripe",
		},
	)
	if err != nil {
		t.Fatalf("create subscription checkout: %v", err)
	}

	failed, err := service.Fail(
		context.Background(),
		FailRequest{
			OrganizationID:  organizationID,
			PaymentID:       result.Payment.ID,
			ProviderEventID: "evt_subscription_failed",
			FailureCode:     "card_declined",
			OccurredAt:      now.Add(time.Minute),
		},
	)
	if err != nil {
		t.Fatalf("fail subscription checkout: %v", err)
	}
	if failed.Payment.Status != payments.StatusFailed {
		t.Fatalf("payment status = %q, want %q", failed.Payment.Status, payments.StatusFailed)
	}

	assertSubscriptionStatus(t, pool, subscriptionID, subscriptions.StatusPending)
}

func newCheckoutTestService(
	t *testing.T,
) (*Service, *pgxpool.Pool, uuid.UUID, uuid.UUID) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test postgres: %v", err)
	}

	schema := "checkout_test_" + uuid.New().String()
	schema = checkoutStripHyphens(schema)
	quotedSchema := pgx.Identifier{schema}.Sanitize()

	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		admin.Close()
		t.Fatalf("create checkout test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		admin.Close()
		t.Fatalf("parse checkout test database config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		admin.Close()
		t.Fatalf("open checkout test pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(
			context.Background(),
			"DROP SCHEMA "+quotedSchema+" CASCADE",
		)
		admin.Close()
	})

	createCheckoutTestSchema(t, pool)

	organizationID := uuid.New()
	subscriptionID := uuid.New()

	if _, err := pool.Exec(
		ctx,
		`INSERT INTO organizations (id, status)
		 VALUES ($1, 'active')`,
		organizationID,
	); err != nil {
		t.Fatalf("insert checkout organization: %v", err)
	}

	if _, err := pool.Exec(
		ctx,
		`INSERT INTO subscriptions (
			id,
			organization_id,
			plan_id,
			status,
			currency,
			amount_micros,
			interval
		) VALUES ($1, $2, $3, 'pending', 'USD', $4, 'month')`,
		subscriptionID,
		organizationID,
		uuid.New(),
		testSubscriptionAmountMicros,
	); err != nil {
		t.Fatalf("insert checkout subscription: %v", err)
	}

	queries := sqlc.New(pool)

	subscriptionsRepository := subscriptions.NewRepository(queries)
	subscriptionsService := subscriptions.NewService(subscriptionsRepository)

	walletsRepository := wallets.NewRepository(queries)
	walletsService := wallets.NewService(walletsRepository, pool)

	paymentsRepository := payments.NewRepository(queries)
	paymentsService := payments.NewService(paymentsRepository)

	service := NewService(
		paymentsService,
		subscriptionsService,
		walletsService,
	)

	return service, pool, organizationID, subscriptionID
}

func createCheckoutTestSchema(
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
		`CREATE TABLE wallets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID NOT NULL UNIQUE REFERENCES organizations(id),
			currency CHAR(3) NOT NULL DEFAULT 'USD',
			status TEXT NOT NULL DEFAULT 'active',
			balance_micros BIGINT NOT NULL DEFAULT 0,
			reserved_micros BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (id, organization_id),
			CHECK (balance_micros >= 0),
			CHECK (reserved_micros >= 0),
			CHECK (reserved_micros <= balance_micros)
		)`,
		`CREATE TABLE wallet_holds (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID NOT NULL,
			organization_id UUID NOT NULL,
			operation_id UUID NOT NULL UNIQUE,
			amount_micros BIGINT NOT NULL,
			reason TEXT NOT NULL,
			reference_type TEXT,
			reference_id UUID,
			status TEXT NOT NULL DEFAULT 'active',
			captured_at TIMESTAMPTZ,
			released_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			FOREIGN KEY (wallet_id, organization_id)
				REFERENCES wallets (id, organization_id)
		)`,
		`CREATE TABLE wallet_ledger_entries (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID NOT NULL,
			organization_id UUID NOT NULL,
			operation_id UUID NOT NULL UNIQUE,
			direction TEXT NOT NULL,
			reason TEXT NOT NULL,
			amount_micros BIGINT NOT NULL,
			balance_after_micros BIGINT NOT NULL,
			reference_type TEXT,
			reference_id UUID,
			occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			FOREIGN KEY (wallet_id, organization_id)
				REFERENCES wallets (id, organization_id)
		)`,
		`CREATE TABLE payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID NOT NULL REFERENCES organizations(id),
			purpose TEXT NOT NULL,
			provider TEXT NOT NULL,
			subscription_id UUID,
			amount_micros BIGINT NOT NULL,
			currency CHAR(3) NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			provider_reference TEXT,
			provider_event_id TEXT,
			period_start TIMESTAMPTZ,
			period_end TIMESTAMPTZ,
			failure_code TEXT,
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE UNIQUE INDEX uq_checkout_provider_reference
			ON payments (provider, provider_reference)
			WHERE provider_reference IS NOT NULL`,
		`CREATE UNIQUE INDEX uq_checkout_provider_event
			ON payments (provider, provider_event_id)
			WHERE provider_event_id IS NOT NULL`,
		`CREATE UNIQUE INDEX uq_checkout_pending_subscription
			ON payments (subscription_id)
			WHERE purpose = 'subscription' AND status = 'pending'`,
	}

	for _, statement := range statements {
		if _, err := pool.Exec(
			context.Background(),
			statement,
		); err != nil {
			t.Fatalf("create checkout test schema: %v", err)
		}
	}
}

func assertSubscriptionStatus(
	t *testing.T,
	pool *pgxpool.Pool,
	subscriptionID uuid.UUID,
	want string,
) {
	t.Helper()

	var status string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT status
		 FROM subscriptions
		 WHERE id = $1`,
		subscriptionID,
	).Scan(&status); err != nil {
		t.Fatalf("read subscription status: %v", err)
	}
	if status != want {
		t.Fatalf("subscription status = %q, want %q", status, want)
	}
}

func checkoutStripHyphens(value string) string {
	result := make([]byte, 0, len(value))
	for index := range len(value) {
		if value[index] != '-' {
			result = append(result, value[index])
		}
	}

	return string(result)
}

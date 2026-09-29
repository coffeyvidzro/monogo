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

func TestCreateSubscriptionCheckoutSnapshotsPurchase(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	result, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create subscription checkout: %v", err)
	}

	if result.Purpose != PurposeSubscription {
		t.Fatalf("purpose = %q, want %q", result.Purpose, PurposeSubscription)
	}
	if result.Status != StatusPending {
		t.Fatalf("status = %q, want %q", result.Status, StatusPending)
	}
	if result.AmountMicros != testSubscriptionAmountMicros {
		t.Fatalf("amount = %d, want %d", result.AmountMicros, testSubscriptionAmountMicros)
	}
	if result.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", result.Currency)
	}
	if result.SubscriptionID == nil || *result.SubscriptionID != subscriptionID {
		t.Fatalf("subscription id = %v, want %s", result.SubscriptionID, subscriptionID)
	}
	if result.Reference == "" {
		t.Fatal("reference is empty")
	}
	if !result.ExpiresAt.Equal(now.Add(checkoutTTL)) {
		t.Fatalf("expires_at = %v, want %v", result.ExpiresAt, now.Add(checkoutTTL))
	}

	assertSubscriptionStatus(t, pool, subscriptionID, subscriptions.StatusPending)
}

func TestConfirmCheckoutCreatesInternalPaymentAttempt(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}

	confirmed, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	)
	if err != nil {
		t.Fatalf("confirm checkout: %v", err)
	}

	if confirmed.Status != StatusProcessing {
		t.Fatalf("status = %q, want %q", confirmed.Status, StatusProcessing)
	}
	if confirmed.Provider == nil || *confirmed.Provider != ProviderStripe {
		t.Fatalf("provider = %v, want %q", confirmed.Provider, ProviderStripe)
	}
	if confirmed.PaymentMethod == nil || *confirmed.PaymentMethod != PaymentMethodCard {
		t.Fatalf("payment method = %v, want %q", confirmed.PaymentMethod, PaymentMethodCard)
	}

	var (
		attempt       int32
		amount        int64
		provider      string
		paymentMethod string
	)
	if err := pool.QueryRow(
		context.Background(),
		`SELECT attempt, amount_micros, provider, payment_method
		 FROM payments
		 WHERE checkout_id = $1`,
		checkout.ID,
	).Scan(&attempt, &amount, &provider, &paymentMethod); err != nil {
		t.Fatalf("read payment attempt: %v", err)
	}
	if attempt != 1 {
		t.Fatalf("attempt = %d, want 1", attempt)
	}
	if amount != testSubscriptionAmountMicros {
		t.Fatalf("payment amount = %d, want %d", amount, testSubscriptionAmountMicros)
	}
	if provider != ProviderStripe {
		t.Fatalf("payment provider = %q, want %q", provider, ProviderStripe)
	}
	if paymentMethod != PaymentMethodCard {
		t.Fatalf("payment method = %q, want %q", paymentMethod, PaymentMethodCard)
	}

	var paymentCount int
	if err := pool.QueryRow(
		context.Background(),
		`SELECT count(*) FROM payments WHERE checkout_id = $1`,
		checkout.ID,
	).Scan(&paymentCount); err != nil {
		t.Fatalf("count payment attempts: %v", err)
	}
	if paymentCount != 1 {
		t.Fatalf("payment attempts = %d, want 1", paymentCount)
	}
}

func TestContinueCheckoutRequiresMatchingActionAndReturnsToWait(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}
	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	); err != nil {
		t.Fatalf("confirm checkout: %v", err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE checkouts
		 SET next_action = 'submit_otp',
		     provider_message = 'Enter the OTP'
		 WHERE id = $1`,
		checkout.ID,
	); err != nil {
		t.Fatalf("set checkout continuation action: %v", err)
	}

	phone := "+233201234567"
	if _, err := service.Continue(
		context.Background(),
		ContinueRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Action:         ActionSubmitPhone,
			Phone:          &phone,
		},
	); err == nil {
		t.Fatal("continue checkout with mismatched action succeeded")
	}

	var nextAction string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT next_action FROM checkouts WHERE id = $1`,
		checkout.ID,
	).Scan(&nextAction); err != nil {
		t.Fatalf("read checkout action after mismatch: %v", err)
	}
	if nextAction != ActionSubmitOTP {
		t.Fatalf("next_action after mismatch = %q, want %q", nextAction, ActionSubmitOTP)
	}

	otp := "123456"
	continued, err := service.Continue(
		context.Background(),
		ContinueRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Action:         ActionSubmitOTP,
			OTP:            &otp,
		},
	)
	if err != nil {
		t.Fatalf("continue checkout: %v", err)
	}
	if continued.NextAction != ActionWait {
		t.Fatalf("next_action = %q, want %q", continued.NextAction, ActionWait)
	}
	if continued.ProviderMessage != nil {
		t.Fatalf("provider_message = %q, want nil", *continued.ProviderMessage)
	}
}

func TestConfirmCheckoutRollsBackOnConflictingPaymentAttempt(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`INSERT INTO payments (
			checkout_id,
			organization_id,
			provider,
			payment_method,
			attempt,
			amount_micros,
			currency,
			status
		) VALUES ($1, $2, 'paystack', 'mobile_money', 1, $3, 'USD', 'pending')`,
		checkout.ID,
		organizationID,
		testSubscriptionAmountMicros,
	); err != nil {
		t.Fatalf("insert conflicting payment attempt: %v", err)
	}

	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	); err == nil {
		t.Fatal("confirm checkout with conflicting payment attempt succeeded")
	}

	var (
		status        string
		provider      *string
		paymentMethod *string
	)
	if err := pool.QueryRow(
		context.Background(),
		`SELECT status, provider, payment_method
		 FROM checkouts
		 WHERE id = $1`,
		checkout.ID,
	).Scan(&status, &provider, &paymentMethod); err != nil {
		t.Fatalf("read checkout after rollback: %v", err)
	}
	if status != StatusPending {
		t.Fatalf("checkout status = %q, want %q", status, StatusPending)
	}
	if provider != nil || paymentMethod != nil {
		t.Fatalf("checkout payment binding was not rolled back: provider=%v method=%v", provider, paymentMethod)
	}

	var paymentCount int
	if err := pool.QueryRow(
		context.Background(),
		`SELECT count(*) FROM payments WHERE checkout_id = $1`,
		checkout.ID,
	).Scan(&paymentCount); err != nil {
		t.Fatalf("count payment attempts: %v", err)
	}
	if paymentCount != 1 {
		t.Fatalf("payment attempts = %d, want existing conflicting attempt only", paymentCount)
	}
}

func TestConfirmCheckoutRollsBackWhenActivePaymentConflicts(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`INSERT INTO payments (
			checkout_id,
			organization_id,
			provider,
			payment_method,
			attempt,
			amount_micros,
			currency
		) VALUES ($1, $2, 'paystack', 'mobile_money', 1, $3, 'USD')`,
		checkout.ID,
		organizationID,
		testSubscriptionAmountMicros,
	); err != nil {
		t.Fatalf("insert conflicting payment attempt: %v", err)
	}

	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	); err == nil {
		t.Fatal("confirm checkout succeeded with conflicting active payment")
	}

	var (
		status        string
		provider      *string
		paymentMethod *string
	)
	if err := pool.QueryRow(
		context.Background(),
		`SELECT status, provider, payment_method
		 FROM checkouts
		 WHERE id = $1`,
		checkout.ID,
	).Scan(&status, &provider, &paymentMethod); err != nil {
		t.Fatalf("read checkout after failed confirmation: %v", err)
	}
	if status != StatusPending {
		t.Fatalf("status = %q, want %q", status, StatusPending)
	}
	if provider != nil || paymentMethod != nil {
		t.Fatalf("payment binding persisted after rollback: provider=%v method=%v", provider, paymentMethod)
	}
}

func TestContinueRejectsStaleAction(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout := createConfirmedCheckoutForContinuation(
		t,
		service,
		pool,
		organizationID,
		subscriptionID,
		ActionSubmitOTP,
	)

	phone := "+233201234567"
	if _, err := service.Continue(
		context.Background(),
		ContinueRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Action:         ActionSubmitPhone,
			Phone:          &phone,
		},
	); err == nil {
		t.Fatal("stale continuation action succeeded")
	}

	assertCheckoutNextAction(t, pool, checkout.ID, ActionSubmitOTP)
}

func TestContinueAuthorizeMobileMoneyAdvancesToWait(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout := createConfirmedCheckoutForContinuation(
		t,
		service,
		pool,
		organizationID,
		subscriptionID,
		ActionAuthorizeMobileMoney,
	)

	continued, err := service.Continue(
		context.Background(),
		ContinueRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Action:         ActionAuthorizeMobileMoney,
		},
	)
	if err != nil {
		t.Fatalf("continue mobile money authorization: %v", err)
	}
	if continued.NextAction != ActionWait {
		t.Fatalf("next action = %q, want %q", continued.NextAction, ActionWait)
	}

	assertCheckoutNextAction(t, pool, checkout.ID, ActionWait)
}

func TestContinueOTPDoesNotAdvanceWithoutProviderAdapter(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout := createConfirmedCheckoutForContinuation(
		t,
		service,
		pool,
		organizationID,
		subscriptionID,
		ActionSubmitOTP,
	)

	otp := "123456"
	if _, err := service.Continue(
		context.Background(),
		ContinueRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Action:         ActionSubmitOTP,
			OTP:            &otp,
		},
	); err == nil {
		t.Fatal("otp continuation succeeded without provider adapter")
	}

	assertCheckoutNextAction(t, pool, checkout.ID, ActionSubmitOTP)
}

func TestCompleteSubscriptionCheckoutActivatesAndReplays(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}
	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	); err != nil {
		t.Fatalf("confirm checkout: %v", err)
	}

	completedAt := now.Add(time.Minute)
	completed, err := service.Complete(
		context.Background(),
		organizationID,
		checkout.ID,
		completedAt,
	)
	if err != nil {
		t.Fatalf("complete checkout: %v", err)
	}
	if completed.Status != StatusSucceeded {
		t.Fatalf("status = %q, want %q", completed.Status, StatusSucceeded)
	}

	replayed, err := service.Complete(
		context.Background(),
		organizationID,
		checkout.ID,
		completedAt,
	)
	if err != nil {
		t.Fatalf("replay checkout completion: %v", err)
	}
	if replayed.ID != completed.ID {
		t.Fatalf("replayed checkout id = %s, want %s", replayed.ID, completed.ID)
	}

	assertSubscriptionStatus(t, pool, subscriptionID, subscriptions.StatusActive)
}

func TestCompleteWalletTopupCreditsOnceAndReplays(t *testing.T) {
	service, pool, organizationID, _ := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeWalletTopup,
			AmountMicros:   50_000_000,
		},
	)
	if err != nil {
		t.Fatalf("create wallet top-up checkout: %v", err)
	}
	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderPaystack,
			PaymentMethod:  PaymentMethodMobileMoney,
		},
	); err != nil {
		t.Fatalf("confirm wallet top-up checkout: %v", err)
	}

	completedAt := now.Add(time.Minute)
	if _, err := service.Complete(
		context.Background(),
		organizationID,
		checkout.ID,
		completedAt,
	); err != nil {
		t.Fatalf("complete wallet top-up checkout: %v", err)
	}
	if _, err := service.Complete(
		context.Background(),
		organizationID,
		checkout.ID,
		completedAt,
	); err != nil {
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
		checkout.ID,
	).Scan(&ledgerCount); err != nil {
		t.Fatalf("count top-up ledger entries: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("ledger entries = %d, want 1", ledgerCount)
	}
}

func TestFailedCheckoutDoesNotApplySubscription(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}
	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	); err != nil {
		t.Fatalf("confirm checkout: %v", err)
	}

	message := "card declined"
	failed, err := service.Fail(
		context.Background(),
		organizationID,
		checkout.ID,
		"card_declined",
		&message,
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("fail checkout: %v", err)
	}
	if failed.Status != StatusFailed {
		t.Fatalf("status = %q, want %q", failed.Status, StatusFailed)
	}

	assertSubscriptionStatus(t, pool, subscriptionID, subscriptions.StatusPending)
}

func TestExpireDueCheckoutReleasesSubscriptionSlot(t *testing.T) {
	service, pool, organizationID, subscriptionID := newCheckoutTestService(t)
	now := time.Date(2026, time.September, 29, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	first, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create first checkout: %v", err)
	}

	service.now = func() time.Time {
		return now.Add(checkoutTTL + time.Second)
	}
	expired, err := service.ExpireDue(context.Background(), 100)
	if err != nil {
		t.Fatalf("expire due checkouts: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired count = %d, want 1", expired)
	}

	var status string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT status FROM checkouts WHERE id = $1`,
		first.ID,
	).Scan(&status); err != nil {
		t.Fatalf("read expired checkout: %v", err)
	}
	if status != StatusExpired {
		t.Fatalf("checkout status = %q, want %q", status, StatusExpired)
	}

	second, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create replacement checkout: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("replacement checkout reused expired checkout id %s", first.ID)
	}
}

func createConfirmedCheckoutForContinuation(
	t *testing.T,
	service *Service,
	pool *pgxpool.Pool,
	organizationID uuid.UUID,
	subscriptionID uuid.UUID,
	nextAction string,
) Checkout {
	t.Helper()

	checkout, err := service.Create(
		context.Background(),
		CreateRequest{
			OrganizationID: organizationID,
			Purpose:        PurposeSubscription,
			SubscriptionID: &subscriptionID,
		},
	)
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}
	if _, err := service.Confirm(
		context.Background(),
		ConfirmRequest{
			OrganizationID: organizationID,
			CheckoutID:     checkout.ID,
			Provider:       ProviderStripe,
			PaymentMethod:  PaymentMethodCard,
		},
	); err != nil {
		t.Fatalf("confirm checkout: %v", err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE checkouts
		 SET next_action = $1,
		     provider_message = 'provider action required'
		 WHERE id = $2`,
		nextAction,
		checkout.ID,
	); err != nil {
		t.Fatalf("set checkout continuation action: %v", err)
	}

	checkout.NextAction = nextAction
	return checkout
}

func assertCheckoutNextAction(
	t *testing.T,
	pool *pgxpool.Pool,
	checkoutID uuid.UUID,
	want string,
) {
	t.Helper()

	var action string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT next_action
		 FROM checkouts
		 WHERE id = $1`,
		checkoutID,
	).Scan(&action); err != nil {
		t.Fatalf("read checkout next action: %v", err)
	}
	if action != want {
		t.Fatalf("next action = %q, want %q", action, want)
	}
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

	schema := checkoutStripHyphens("checkout_test_" + uuid.New().String())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		admin.Close()
		t.Fatalf("create checkout test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatalf("parse checkout test database config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
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
		t.Fatalf("insert organization: %v", err)
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
		t.Fatalf("insert subscription: %v", err)
	}

	queries := sqlc.New(pool)
	subscriptionsService := subscriptions.NewService(
		subscriptions.NewRepository(queries),
	)
	walletsService := wallets.NewService(
		wallets.NewRepository(queries),
		pool,
	)
	paymentsService := payments.NewService(
		payments.NewRepository(queries),
	)
	service := NewService(
		NewRepository(queries),
		paymentsService,
		subscriptionsService,
		walletsService,
		pool,
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
		`CREATE TABLE checkouts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID NOT NULL REFERENCES organizations(id),
			purpose TEXT NOT NULL,
			subscription_id UUID,
			reference TEXT NOT NULL UNIQUE,
			amount_micros BIGINT NOT NULL,
			currency CHAR(3) NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			provider TEXT,
			payment_method TEXT,
			next_action TEXT NOT NULL DEFAULT 'wait',
			provider_message TEXT,
			failure_code TEXT,
			expires_at TIMESTAMPTZ NOT NULL,
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (id, organization_id)
		)`,
		`CREATE UNIQUE INDEX uq_checkout_test_active_subscription
			ON checkouts (subscription_id)
			WHERE purpose = 'subscription'
			  AND status IN ('pending', 'processing')`,
		`CREATE TABLE payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			checkout_id UUID NOT NULL,
			organization_id UUID NOT NULL,
			provider TEXT NOT NULL,
			payment_method TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			provider_payment_id TEXT,
			amount_micros BIGINT NOT NULL,
			currency CHAR(3) NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			failure_code TEXT,
			paid_at TIMESTAMPTZ,
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}

	for _, statement := range statements {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
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

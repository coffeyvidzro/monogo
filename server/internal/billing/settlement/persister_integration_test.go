package settlement

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPersisterFinancialTransactionMatrix(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_TEST_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_TEST_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf(
			"create test pool: %v",
			err,
		)
	}
	defer pool.Close()
	queries := sqlc.New(pool)

	organization, err := queries.CreateOrganization(
		ctx,
		"OCS persistence test "+uuid.NewString(),
	)
	if err != nil {
		t.Fatalf(
			"create organization fixture: %v",
			err,
		)
	}
	wallet, err := queries.CreateBillingWallet(
		ctx,
		sqlc.CreateBillingWalletParams{
			Currency:       "USD",
			OrganizationID: organization.ID,
		},
	)
	if err != nil {
		t.Fatalf(
			"create wallet fixture: %v",
			err,
		)
	}
	charge := createChargeFixture(
		t,
		ctx,
		queries,
		organization.ID,
		wallet.ID,
		"rolling",
	)

	persister := NewPersister(
		NewRepository(
			pool,
			queries,
		),
	)
	occurredAt := time.Now().UTC().Truncate(time.Millisecond)
	events := []redisintegration.OCSEvent{
		{
			StreamID:            "1-0",
			OrganizationID:      organization.ID,
			WalletID:            wallet.ID,
			OperationID:         uuid.New(),
			WalletVersion:       1,
			EventType:           "credit",
			BalanceDeltaMicros:  1_000_000,
			BalanceAfterMicros:  1_000_000,
			ReservedAfterMicros: 0,
			OccurredAt:          occurredAt,
		},
		{
			StreamID:               "2-0",
			OrganizationID:         organization.ID,
			WalletID:               wallet.ID,
			ChargeID:               &charge.ID,
			OperationID:            uuid.New(),
			WalletVersion:          2,
			ChargeSequence:         1,
			EventType:              "reserve",
			ReservedDeltaMicros:    100_000,
			BalanceAfterMicros:     1_000_000,
			ReservedAfterMicros:    100_000,
			ChargeAuthorizedMicros: 100_000,
			ChargeReservedMicros:   100_000,
			ChargeStatus:           "active",
			OccurredAt:             occurredAt.Add(time.Millisecond),
		},
		{
			StreamID:               "3-0",
			OrganizationID:         organization.ID,
			WalletID:               wallet.ID,
			ChargeID:               &charge.ID,
			OperationID:            uuid.New(),
			WalletVersion:          3,
			ChargeSequence:         2,
			EventType:              "consume",
			BalanceDeltaMicros:     -20_000,
			ReservedDeltaMicros:    -20_000,
			BalanceAfterMicros:     980_000,
			ReservedAfterMicros:    80_000,
			ChargeAuthorizedMicros: 100_000,
			ChargeConsumedMicros:   20_000,
			ChargeReservedMicros:   80_000,
			ChargeStatus:           "active",
			OccurredAt:             occurredAt.Add(2 * time.Millisecond),
		},
		{
			StreamID:               "4-0",
			OrganizationID:         organization.ID,
			WalletID:               wallet.ID,
			ChargeID:               &charge.ID,
			OperationID:            uuid.New(),
			WalletVersion:          4,
			ChargeSequence:         3,
			EventType:              "release",
			ReservedDeltaMicros:    -30_000,
			BalanceAfterMicros:     980_000,
			ReservedAfterMicros:    50_000,
			ChargeAuthorizedMicros: 100_000,
			ChargeConsumedMicros:   20_000,
			ChargeReservedMicros:   50_000,
			ChargeStatus:           "active",
			OccurredAt:             occurredAt.Add(3 * time.Millisecond),
		},
		{
			StreamID:               "5-0",
			OrganizationID:         organization.ID,
			WalletID:               wallet.ID,
			ChargeID:               &charge.ID,
			OperationID:            uuid.New(),
			WalletVersion:          5,
			ChargeSequence:         4,
			EventType:              "finalize",
			ReservedDeltaMicros:    -50_000,
			BalanceAfterMicros:     980_000,
			ReservedAfterMicros:    0,
			ChargeAuthorizedMicros: 100_000,
			ChargeConsumedMicros:   20_000,
			ChargeStatus:           "completed",
			OccurredAt:             occurredAt.Add(4 * time.Millisecond),
		},
		{
			StreamID:            "6-0",
			OrganizationID:      organization.ID,
			WalletID:            wallet.ID,
			OperationID:         uuid.New(),
			WalletVersion:       6,
			EventType:           "credit",
			BalanceDeltaMicros:  1_000_000,
			BalanceAfterMicros:  1_980_000,
			ReservedAfterMicros: 0,
			OccurredAt:          occurredAt.Add(5 * time.Millisecond),
		},
	}

	for _, event := range events {
		assertApplied(t, ctx, persister, event)
	}

	duplicateResult, err := persister.Persist(ctx, events[5])
	if err != nil || duplicateResult.Outcome != OutcomeAlreadyApplied {
		t.Fatalf(
			"duplicate outcome = %s, error = %v",
			duplicateResult.Outcome,
			err,
		)
	}

	conflict := events[5]
	conflict.BalanceAfterMicros++
	conflictResult, err := persister.Persist(ctx, conflict)
	if err == nil || conflictResult.Outcome != OutcomeIntegrity {
		t.Fatalf(
			"conflict outcome = %s, error = %v",
			conflictResult.Outcome,
			err,
		)
	}

	gap := events[5]
	gap.StreamID = "8-0"
	gap.OperationID = uuid.New()
	gap.WalletVersion = 8
	gap.BalanceAfterMicros = 2_980_000
	gap.OccurredAt = occurredAt.Add(6 * time.Millisecond)
	gapResult, err := persister.Persist(ctx, gap)
	if err == nil || gapResult.Outcome != OutcomeVersionGap {
		t.Fatalf(
			"gap outcome = %s, error = %v",
			gapResult.Outcome,
			err,
		)
	}

	concurrent := events[5]
	concurrent.StreamID = "7-0"
	concurrent.OperationID = uuid.New()
	concurrent.WalletVersion = 7
	concurrent.BalanceDeltaMicros = 20_000
	concurrent.BalanceAfterMicros = 2_000_000
	concurrent.OccurredAt = occurredAt.Add(7 * time.Millisecond)
	assertConcurrentReplay(t, ctx, persister, concurrent)

	debitCharge := createChargeFixture(
		t,
		ctx,
		queries,
		organization.ID,
		wallet.ID,
		"discrete",
	)
	debit := redisintegration.OCSEvent{
		StreamID:               "8-0",
		OrganizationID:         organization.ID,
		WalletID:               wallet.ID,
		ChargeID:               &debitCharge.ID,
		OperationID:            uuid.New(),
		WalletVersion:          8,
		ChargeSequence:         1,
		EventType:              "debit",
		BalanceDeltaMicros:     -10_000,
		BalanceAfterMicros:     1_990_000,
		ReservedAfterMicros:    0,
		ChargeAuthorizedMicros: 10_000,
		ChargeConsumedMicros:   10_000,
		ChargeStatus:           "active",
		OccurredAt:             occurredAt.Add(8 * time.Millisecond),
	}
	assertApplied(t, ctx, persister, debit)

	assertFinancialState(
		t,
		ctx,
		queries,
		organization.ID,
		wallet.ID,
		charge.ID,
		1_990_000,
		0,
		8,
		100_000,
		20_000,
		0,
		"completed",
		8,
		5,
	)
	assertChargeState(
		t,
		ctx,
		queries,
		organization.ID,
		debitCharge.ID,
		10_000,
		10_000,
		0,
		"active",
	)
}

func createChargeFixture(
	t *testing.T,
	ctx context.Context,
	queries *sqlc.Queries,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	mode string,
) sqlc.Charge {
	t.Helper()

	params := sqlc.CreateBillingChargeParams{
		OrganizationID: organizationID,
		WalletID:       walletID,
		ResourceType:   "test_resource",
		ResourceID:     uuid.New(),
		ChargingMode:   mode,
		Currency:       "USD",
		IdempotencyKey: uuid.NewString(),
		RequestHash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PricingSnapshot: json.RawMessage{
			'{',
			'}',
		},
	}
	charge, err := queries.CreateBillingCharge(
		ctx,
		params,
	)
	if err != nil {
		t.Fatalf(
			"create charge fixture: %v",
			err,
		)
	}

	return charge
}

func assertApplied(
	t *testing.T,
	ctx context.Context,
	persister *Persister,
	event redisintegration.OCSEvent,
) {
	t.Helper()

	result, err := persister.Persist(ctx, event)
	if err != nil || result.Outcome != OutcomeApplied {
		t.Fatalf(
			"persist %s outcome = %s, error = %v",
			event.EventType,
			result.Outcome,
			err,
		)
	}
}

func assertConcurrentReplay(
	t *testing.T,
	ctx context.Context,
	persister *Persister,
	event redisintegration.OCSEvent,
) {
	t.Helper()

	outcomes := make(chan Outcome, 2)
	errorsChannel := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := persister.Persist(ctx, event)
			outcomes <- result.Outcome
			errorsChannel <- err
		}()
	}
	group.Wait()
	close(outcomes)
	close(errorsChannel)

	applied := 0
	alreadyApplied := 0
	for outcome := range outcomes {
		switch outcome {
		case OutcomeApplied:
			applied++
		case OutcomeAlreadyApplied:
			alreadyApplied++
		}
	}
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf(
				"concurrent persistence: %v",
				err,
			)
		}
	}
	if applied != 1 || alreadyApplied != 1 {
		t.Fatalf(
			"concurrent outcomes applied=%d already_applied=%d",
			applied,
			alreadyApplied,
		)
	}
}

func assertFinancialState(
	t *testing.T,
	ctx context.Context,
	queries *sqlc.Queries,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	chargeID uuid.UUID,
	balance int64,
	reserved int64,
	version int64,
	authorized int64,
	consumed int64,
	chargeReserved int64,
	status string,
	walletEventCount int,
	ledgerCount int,
) {
	t.Helper()

	wallet, err := queries.GetBillingWalletByID(
		ctx,
		sqlc.GetBillingWalletByIDParams{
			ID:             walletID,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		t.Fatalf(
			"read wallet state: %v",
			err,
		)
	}
	if wallet.BalanceMicros != balance ||
		wallet.ReservedMicros != reserved ||
		wallet.OcsVersion != version {
		t.Fatalf(
			"wallet state = (%d, %d, %d), want (%d, %d, %d)",
			wallet.BalanceMicros,
			wallet.ReservedMicros,
			wallet.OcsVersion,
			balance,
			reserved,
			version,
		)
	}

	assertChargeState(
		t,
		ctx,
		queries,
		organizationID,
		chargeID,
		authorized,
		consumed,
		chargeReserved,
		status,
	)

	walletEvents, err := queries.ListWalletEvents(
		ctx,
		sqlc.ListWalletEventsParams{
			WalletID:       walletID,
			OrganizationID: organizationID,
			PageLimit:      100,
		},
	)
	if err != nil {
		t.Fatalf(
			"list wallet events: %v",
			err,
		)
	}
	ledgerEntries, err := queries.ListWalletLedgerEntries(
		ctx,
		sqlc.ListWalletLedgerEntriesParams{
			WalletID:       walletID,
			OrganizationID: organizationID,
			PageLimit:      100,
		},
	)
	if err != nil {
		t.Fatalf(
			"list ledger entries: %v",
			err,
		)
	}
	if len(walletEvents) != walletEventCount || len(ledgerEntries) != ledgerCount {
		t.Fatalf(
			"event counts = (%d, %d), want (%d, %d)",
			len(walletEvents),
			len(ledgerEntries),
			walletEventCount,
			ledgerCount,
		)
	}
}

func assertChargeState(
	t *testing.T,
	ctx context.Context,
	queries *sqlc.Queries,
	organizationID uuid.UUID,
	chargeID uuid.UUID,
	authorized int64,
	consumed int64,
	reserved int64,
	status string,
) {
	t.Helper()

	charge, err := queries.GetBillingCharge(
		ctx,
		sqlc.GetBillingChargeParams{
			ID:             chargeID,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		t.Fatalf(
			"read charge state: %v",
			err,
		)
	}
	if charge.AuthorizedMicros != authorized ||
		charge.ConsumedMicros != consumed ||
		charge.ReservedMicros != reserved ||
		charge.Status != status {
		t.Fatalf(
			"charge state = (%d, %d, %d, %s), want (%d, %d, %d, %s)",
			charge.AuthorizedMicros,
			charge.ConsumedMicros,
			charge.ReservedMicros,
			charge.Status,
			authorized,
			consumed,
			reserved,
			status,
		)
	}
}

func TestPersistenceErrorUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("database unavailable")
	err := &PersistenceError{
		Outcome: OutcomeRetry,
		Cause:   cause,
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected persistence error to unwrap its cause")
	}
}

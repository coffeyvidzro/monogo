package wallets

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testWalletBalanceMicros int64 = 1_000_000

func TestHoldReservesAvailableBalance(t *testing.T) {
	service, pool, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	referenceType := "call"
	referenceID := uuid.New()
	req := HoldRequest{
		OrganizationID: organizationID,
		OperationID:    uuid.New(),
		AmountMicros:   400_000,
		Reason:         "managed_voice",
		ReferenceType:  &referenceType,
		ReferenceID:    &referenceID,
	}

	hold, err := service.Hold(
		context.Background(),
		req,
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}
	if hold.Status != HoldStatusActive {
		t.Fatalf("hold status = %q, want %q", hold.Status, HoldStatusActive)
	}

	replayed, err := service.Hold(
		context.Background(),
		req,
	)
	if err != nil {
		t.Fatalf("replay hold: %v", err)
	}
	if replayed.ID != hold.ID {
		t.Fatalf("replayed hold id = %s, want %s", replayed.ID, hold.ID)
	}

	wallet, err := service.Get(
		context.Background(),
		organizationID,
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	assertWalletAmounts(
		t,
		wallet,
		testWalletBalanceMicros,
		400_000,
		600_000,
	)

	assertLedgerCount(
		t,
		pool,
		0,
	)
}

func TestHoldRejectsUnavailableBalance(t *testing.T) {
	service, _, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    uuid.New(),
			AmountMicros:   700_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("first hold: %v", err)
	}

	_, err = service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    uuid.New(),
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	requireAppErrorCode(
		t,
		err,
		"PAYMENT_REQUIRED",
	)
}

func TestDirectDebitCannotSpendReservedBalance(t *testing.T) {
	service, _, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    uuid.New(),
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	_, err = service.Debit(
		context.Background(),
		MovementRequest{
			OrganizationID: organizationID,
			OperationID:    uuid.New(),
			AmountMicros:   700_000,
			Reason:         "managed_number",
		},
	)
	requireAppErrorCode(
		t,
		err,
		"PAYMENT_REQUIRED",
	)

	_, err = service.Debit(
		context.Background(),
		MovementRequest{
			OrganizationID: organizationID,
			OperationID:    uuid.New(),
			AmountMicros:   600_000,
			Reason:         "managed_number",
		},
	)
	if err != nil {
		t.Fatalf("debit available balance: %v", err)
	}

	wallet, err := service.Get(
		context.Background(),
		organizationID,
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	assertWalletAmounts(
		t,
		wallet,
		400_000,
		400_000,
		0,
	)
}

func TestCaptureDebitsOnce(t *testing.T) {
	service, pool, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	operationID := uuid.New()
	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	entry, err := service.Capture(
		context.Background(),
		organizationID,
		operationID,
	)
	if err != nil {
		t.Fatalf("capture hold: %v", err)
	}
	if entry.Direction != DirectionDebit {
		t.Fatalf("ledger direction = %q, want %q", entry.Direction, DirectionDebit)
	}
	if entry.AmountMicros != 400_000 {
		t.Fatalf("ledger amount = %d, want %d", entry.AmountMicros, 400_000)
	}

	replayed, err := service.Capture(
		context.Background(),
		organizationID,
		operationID,
	)
	if err != nil {
		t.Fatalf("replay capture: %v", err)
	}
	if replayed.ID != entry.ID {
		t.Fatalf("replayed ledger id = %s, want %s", replayed.ID, entry.ID)
	}

	wallet, err := service.Get(
		context.Background(),
		organizationID,
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	assertWalletAmounts(
		t,
		wallet,
		600_000,
		0,
		600_000,
	)

	assertLedgerCount(
		t,
		pool,
		1,
	)
}

func TestCaptureAmountDebitsUsageAndReleasesUnusedAuthorization(t *testing.T) {
	service, _, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)
	operationID := uuid.New()
	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   600_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	entry, err := service.CaptureAmount(
		context.Background(),
		organizationID,
		operationID,
		200_000,
	)
	if err != nil {
		t.Fatalf("capture partial hold: %v", err)
	}
	if entry.AmountMicros != 200_000 {
		t.Fatalf("ledger amount = %d, want %d", entry.AmountMicros, 200_000)
	}

	wallet, err := service.Get(
		context.Background(),
		organizationID,
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	assertWalletAmounts(
		t,
		wallet,
		800_000,
		0,
		800_000,
	)

	_, err = service.CaptureAmount(
		context.Background(),
		organizationID,
		operationID,
		300_000,
	)
	requireAppErrorCode(
		t,
		err,
		"CONFLICT",
	)
}

func TestReleaseRestoresAvailableBalanceWithoutLedgerEntry(t *testing.T) {
	service, pool, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	operationID := uuid.New()
	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	hold, err := service.Release(
		context.Background(),
		organizationID,
		operationID,
	)
	if err != nil {
		t.Fatalf("release hold: %v", err)
	}
	if hold.Status != HoldStatusReleased {
		t.Fatalf("hold status = %q, want %q", hold.Status, HoldStatusReleased)
	}

	replayed, err := service.Release(
		context.Background(),
		organizationID,
		operationID,
	)
	if err != nil {
		t.Fatalf("replay release: %v", err)
	}
	if replayed.ID != hold.ID {
		t.Fatalf("replayed hold id = %s, want %s", replayed.ID, hold.ID)
	}

	wallet, err := service.Get(
		context.Background(),
		organizationID,
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	assertWalletAmounts(
		t,
		wallet,
		testWalletBalanceMicros,
		0,
		testWalletBalanceMicros,
	)

	assertLedgerCount(
		t,
		pool,
		0,
	)
}

func TestCapturedHoldCannotBeReleased(t *testing.T) {
	service, _, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	operationID := uuid.New()
	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	if _, err := service.Capture(
		context.Background(),
		organizationID,
		operationID,
	); err != nil {
		t.Fatalf("capture hold: %v", err)
	}

	_, err = service.Release(
		context.Background(),
		organizationID,
		operationID,
	)
	requireAppErrorCode(
		t,
		err,
		"CONFLICT",
	)
}

func TestReleasedHoldCannotBeCaptured(t *testing.T) {
	service, _, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	operationID := uuid.New()
	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	if _, err := service.Release(
		context.Background(),
		organizationID,
		operationID,
	); err != nil {
		t.Fatalf("release hold: %v", err)
	}

	_, err = service.Capture(
		context.Background(),
		organizationID,
		operationID,
	)
	requireAppErrorCode(
		t,
		err,
		"CONFLICT",
	)
}

func TestConcurrentHoldsCannotOverspendWallet(t *testing.T) {
	service, pool, organizationID := newWalletTestService(
		t,
		500_000,
	)

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			_, err := service.Hold(
				context.Background(),
				HoldRequest{
					OrganizationID: organizationID,
					OperationID:    uuid.New(),
					AmountMicros:   500_000,
					Reason:         "managed_voice",
				},
			)

			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var successes int
	var paymentRequired int

	for err := range results {
		if err == nil {
			successes++
			continue
		}

		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == "PAYMENT_REQUIRED" {
			paymentRequired++
			continue
		}

		t.Fatalf("unexpected concurrent hold error: %v", err)
	}

	if successes != 1 {
		t.Fatalf("successful holds = %d, want 1", successes)
	}
	if paymentRequired != 1 {
		t.Fatalf("payment-required holds = %d, want 1", paymentRequired)
	}

	wallet, err := service.Get(
		context.Background(),
		organizationID,
	)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	assertWalletAmounts(
		t,
		wallet,
		500_000,
		500_000,
		0,
	)

	var activeHolds int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM wallet_holds WHERE status = 'active'",
	).Scan(&activeHolds); err != nil {
		t.Fatalf("count active holds: %v", err)
	}
	if activeHolds != 1 {
		t.Fatalf("active holds = %d, want 1", activeHolds)
	}
}

func TestHoldReplayWithDifferentAmountConflicts(t *testing.T) {
	service, _, organizationID := newWalletTestService(
		t,
		testWalletBalanceMicros,
	)

	operationID := uuid.New()
	_, err := service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   400_000,
			Reason:         "managed_voice",
		},
	)
	if err != nil {
		t.Fatalf("hold funds: %v", err)
	}

	_, err = service.Hold(
		context.Background(),
		HoldRequest{
			OrganizationID: organizationID,
			OperationID:    operationID,
			AmountMicros:   500_000,
			Reason:         "managed_voice",
		},
	)
	requireAppErrorCode(
		t,
		err,
		"CONFLICT",
	)
}

func newWalletTestService(
	t *testing.T,
	balanceMicros int64,
) (*Service, *pgxpool.Pool, uuid.UUID) {
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

	schema := "wallet_test_" + uuid.New().String()
	schema = stringsWithoutHyphens(schema)
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
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		admin.Close()
		t.Fatalf("parse test database config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(
		ctx,
		config,
	)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
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

	createWalletTestSchema(
		t,
		pool,
	)

	organizationID := uuid.New()
	walletID := uuid.New()

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

	if _, err := pool.Exec(
		ctx,
		`INSERT INTO wallets (
			id,
			organization_id,
			balance_micros
		) VALUES ($1, $2, $3)`,
		walletID,
		organizationID,
		balanceMicros,
	); err != nil {
		t.Fatalf("insert test wallet: %v", err)
	}

	queries := sqlc.New(pool)
	repository := NewRepository(queries)
	service := NewService(
		repository,
		pool,
	)

	return service, pool, organizationID
}

func createWalletTestSchema(
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
		`CREATE TABLE wallets (
			id UUID PRIMARY KEY,
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
	}

	for _, statement := range statements {
		if _, err := pool.Exec(
			context.Background(),
			statement,
		); err != nil {
			t.Fatalf("create wallet test schema: %v", err)
		}
	}
}

func assertWalletAmounts(
	t *testing.T,
	wallet Wallet,
	balanceMicros int64,
	reservedMicros int64,
	availableMicros int64,
) {
	t.Helper()

	if wallet.BalanceMicros != balanceMicros {
		t.Fatalf(
			"wallet balance = %d, want %d",
			wallet.BalanceMicros,
			balanceMicros,
		)
	}
	if wallet.ReservedMicros != reservedMicros {
		t.Fatalf(
			"wallet reserved = %d, want %d",
			wallet.ReservedMicros,
			reservedMicros,
		)
	}
	if wallet.AvailableMicros != availableMicros {
		t.Fatalf(
			"wallet available = %d, want %d",
			wallet.AvailableMicros,
			availableMicros,
		)
	}
}

func assertLedgerCount(
	t *testing.T,
	pool *pgxpool.Pool,
	want int,
) {
	t.Helper()

	var count int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM wallet_ledger_entries",
	).Scan(&count); err != nil {
		t.Fatalf("count wallet ledger entries: %v", err)
	}
	if count != want {
		t.Fatalf(
			"wallet ledger entries = %d, want %d",
			count,
			want,
		)
	}
}

func requireAppErrorCode(
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

func stringsWithoutHyphens(value string) string {
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

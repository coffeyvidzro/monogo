package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOCSStreamPendingRecovery(t *testing.T) {
	redisURL := os.Getenv("REDIS_TEST_URL")
	if redisURL == "" {
		t.Skip("REDIS_TEST_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := New(
		ctx,
		DefaultConfig(redisURL),
	)
	if err != nil {
		t.Fatalf(
			"create Redis client: %v",
			err,
		)
	}
	defer func() {
		_ = client.Close()
	}()

	ocs, err := client.NewOCS()
	if err != nil {
		t.Fatalf(
			"create OCS client: %v",
			err,
		)
	}
	organizationID := uuid.New()
	walletID := uuid.New()
	operationID := uuid.New()
	group := "test-" + uuid.NewString()

	if _, err := ocs.EnsureWallet(
		ctx,
		OCSWalletSeed{
			OrganizationID: organizationID,
			WalletID:       walletID,
			Currency:       "USD",
			Status:         "active",
			BalanceMicros:  0,
			ReservedMicros: 0,
			Version:        0,
		},
	); err != nil {
		t.Fatalf(
			"ensure wallet: %v",
			err,
		)
	}
	if err := ocs.EnsureConsumerGroup(ctx, group); err != nil {
		t.Fatalf(
			"ensure consumer group: %v",
			err,
		)
	}
	if _, err := ocs.Credit(
		ctx,
		OCSCreditRequest{
			OrganizationID: organizationID,
			WalletID:       walletID,
			Currency:       "USD",
			OperationID:    operationID,
			AmountMicros:   1_000_000,
		},
	); err != nil {
		t.Fatalf(
			"credit wallet: %v",
			err,
		)
	}

	events, err := ocs.ReadGroup(
		ctx,
		group,
		"original-consumer",
		1,
		time.Second,
	)
	if err != nil {
		t.Fatalf(
			"read stream event: %v",
			err,
		)
	}
	if len(events) != 1 || events[0].OperationID != operationID {
		t.Fatalf(
			"read events = %+v, want operation %s",
			events,
			operationID,
		)
	}

	recovered, _, err := ocs.RecoverPending(
		ctx,
		group,
		"recovery-consumer",
		0,
		"0-0",
		1,
	)
	if err != nil {
		t.Fatalf(
			"recover pending event: %v",
			err,
		)
	}
	if len(recovered) != 1 || recovered[0].StreamID != events[0].StreamID {
		t.Fatalf(
			"recovered events = %+v, want stream %s",
			recovered,
			events[0].StreamID,
		)
	}
	if err := ocs.Acknowledge(ctx, group, recovered[0].StreamID); err != nil {
		t.Fatalf(
			"acknowledge recovered event: %v",
			err,
		)
	}

	afterAck, _, err := ocs.RecoverPending(
		ctx,
		group,
		"recovery-consumer",
		0,
		"0-0",
		1,
	)
	if err != nil {
		t.Fatalf(
			"inspect acknowledged event: %v",
			err,
		)
	}
	if len(afterAck) != 0 {
		t.Fatalf(
			"pending events after acknowledgement = %d, want 0",
			len(afterAck),
		)
	}
}

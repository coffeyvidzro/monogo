package wallets

import (
	"context"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type expirationRepoStub struct{ rows []sqlc.WalletReservation }

func (s expirationRepoStub) ExpiredReservations(context.Context, int32) ([]sqlc.WalletReservation, error) {
	return s.rows, nil
}

type expirationServiceStub struct{ ids []uuid.UUID }

func (s *expirationServiceStub) Expire(_ context.Context, _ uuid.UUID, id uuid.UUID) (sqlc.WalletReservation, error) {
	s.ids = append(s.ids, id)
	return sqlc.WalletReservation{ID: id}, nil
}

func TestExpirationJobExpiresCandidates(t *testing.T) {
	first, second, organizationID := uuid.New(), uuid.New(), uuid.New()
	service := &expirationServiceStub{}
	job, err := NewExpirationJob(expirationRepoStub{rows: []sqlc.WalletReservation{{ID: first, OrganizationID: organizationID}, {ID: second, OrganizationID: organizationID}}}, service, 10, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(service.ids) != 2 || service.ids[0] != first || service.ids[1] != second {
		t.Fatalf("expired IDs = %v", service.ids)
	}
}

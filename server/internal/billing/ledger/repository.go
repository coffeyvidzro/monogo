package ledger

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	if queries == nil {
		panic("billing ledger: queries are required")
	}

	return &Repository{
		queries: queries,
	}
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.WalletLedgerEntry, error) {
	params := sqlc.GetWalletLedgerEntryParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.GetWalletLedgerEntry(
		ctx,
		params,
	)
}

func (r *Repository) ListWallet(
	ctx context.Context,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletLedgerEntry, error) {
	params := sqlc.ListWalletLedgerEntriesParams{
		WalletID:       walletID,
		OrganizationID: organizationID,
		PageLimit:      limit,
	}

	return r.queries.ListWalletLedgerEntries(
		ctx,
		params,
	)
}

func (r *Repository) ListCharge(
	ctx context.Context,
	organizationID uuid.UUID,
	chargeID uuid.UUID,
) ([]sqlc.WalletLedgerEntry, error) {
	params := sqlc.ListChargeLedgerEntriesParams{
		OrganizationID: organizationID,
		ChargeID:       &chargeID,
	}

	return r.queries.ListChargeLedgerEntries(
		ctx,
		params,
	)
}

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
	return &Repository{queries: queries}
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.WalletLedgerEntry, error) {
	return r.queries.GetWalletLedgerEntry(ctx, sqlc.GetWalletLedgerEntryParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) ListWallet(
	ctx context.Context,
	organizationID, walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletLedgerEntry, error) {
	return r.queries.ListWalletLedgerEntries(ctx, sqlc.ListWalletLedgerEntriesParams{
		WalletID:       walletID,
		OrganizationID: organizationID,
		PageLimit:      limit,
	})
}

func (r *Repository) ListCharge(
	ctx context.Context,
	organizationID, chargeID uuid.UUID,
) ([]sqlc.WalletLedgerEntry, error) {
	return r.queries.ListChargeLedgerEntries(ctx, sqlc.ListChargeLedgerEntriesParams{
		OrganizationID: organizationID,
		ChargeID:       &chargeID,
	})
}

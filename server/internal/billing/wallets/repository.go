package wallets

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
		panic("billing wallets: queries are required")
	}
	return &Repository{queries: queries}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	return r.queries.CreateBillingWallet(ctx, sqlc.CreateBillingWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	return r.queries.GetBillingWallet(ctx, sqlc.GetBillingWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
}

func (r *Repository) GetByID(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.GetBillingWalletByID(ctx, sqlc.GetBillingWalletByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Wallet, error) {
	return r.queries.ListBillingWallets(ctx, organizationID)
}

func (r *Repository) SetStatus(
	ctx context.Context,
	organizationID, id uuid.UUID,
	status string,
) (sqlc.Wallet, error) {
	return r.queries.SetBillingWalletStatus(ctx, sqlc.SetBillingWalletStatusParams{
		Status:         status,
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) EventByOperationID(
	ctx context.Context,
	operationID uuid.UUID,
) (sqlc.WalletEvent, error) {
	return r.queries.GetWalletEventByOperationID(ctx, operationID)
}

func (r *Repository) ListEvents(
	ctx context.Context,
	organizationID, walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletEvent, error) {
	return r.queries.ListWalletEvents(ctx, sqlc.ListWalletEventsParams{
		WalletID:       walletID,
		OrganizationID: organizationID,
		PageLimit:      limit,
	})
}

func (r *Repository) ListChargeEvents(
	ctx context.Context,
	organizationID, chargeID uuid.UUID,
) ([]sqlc.WalletEvent, error) {
	return r.queries.ListChargeWalletEvents(ctx, sqlc.ListChargeWalletEventsParams{
		OrganizationID: organizationID,
		ChargeID:       &chargeID,
	})
}

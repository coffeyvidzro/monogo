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

	return &Repository{
		queries: queries,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	params := sqlc.CreateBillingWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	}

	return r.queries.CreateBillingWallet(ctx, params)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	params := sqlc.GetBillingWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	}

	return r.queries.GetBillingWallet(ctx, params)
}

func (r *Repository) GetByID(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Wallet, error) {
	params := sqlc.GetBillingWalletByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.GetBillingWalletByID(ctx, params)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Wallet, error) {
	return r.queries.ListBillingWallets(ctx, organizationID)
}

func (r *Repository) SetStatus(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	status string,
) (sqlc.Wallet, error) {
	params := sqlc.SetBillingWalletStatusParams{
		Status:         status,
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.SetBillingWalletStatus(ctx, params)
}

func (r *Repository) EventByOperationID(
	ctx context.Context,
	operationID uuid.UUID,
) (sqlc.WalletEvent, error) {
	return r.queries.GetWalletEventByOperationID(ctx, operationID)
}

func (r *Repository) ListEvents(
	ctx context.Context,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletEvent, error) {
	params := sqlc.ListWalletEventsParams{
		WalletID:       walletID,
		OrganizationID: organizationID,
		PageLimit:      limit,
	}

	return r.queries.ListWalletEvents(ctx, params)
}

func (r *Repository) ListChargeEvents(
	ctx context.Context,
	organizationID uuid.UUID,
	chargeID uuid.UUID,
) ([]sqlc.WalletEvent, error) {
	params := sqlc.ListChargeWalletEventsParams{
		OrganizationID: organizationID,
		ChargeID:       &chargeID,
	}

	return r.queries.ListChargeWalletEvents(ctx, params)
}

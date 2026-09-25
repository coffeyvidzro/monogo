package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/checkout"
	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/usage"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	Checkout CheckoutModule
	Payments PaymentsModule
	Usage    UsageModule
	Wallets  WalletsModule
}

type CheckoutModule struct {
	Repository *checkout.Repository
	Service    *checkout.Service
}

type PaymentsModule struct {
	Repository *payments.Repository
	Service    *payments.Service
}

type UsageModule struct {
	Repository *usage.Repository
	Service    *usage.Service
}

type WalletsModule struct {
	Repository *wallets.Repository
	Service    *wallets.Service
}

func New(db *pgxpool.Pool, queries *sqlc.Queries) *Module {
	usageRepository := usage.NewRepository(queries)
	return &Module{
		Checkout: CheckoutModule{
			Repository: checkout.NewRepository(db),
			Service:    checkout.NewService(db),
		},
		Payments: PaymentsModule{
			Repository: payments.NewRepository(db),
			Service:    payments.NewService(db),
		},
		Usage: UsageModule{
			Repository: usageRepository,
			Service:    usage.NewService(usageRepository),
		},
		Wallets: WalletsModule{
			Repository: wallets.NewRepository(db),
			Service:    wallets.NewService(db),
		},
	}
}

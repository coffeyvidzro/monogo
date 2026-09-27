package billing

import (
	"github.com/coffeyvidzro/monogo/internal/billing/charges"
	"github.com/coffeyvidzro/monogo/internal/billing/ledger"
	"github.com/coffeyvidzro/monogo/internal/billing/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/billing/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
)

type Module struct {
	Wallets       WalletsModule
	Charges       ChargesModule
	Ledger        LedgerModule
	Subscriptions SubscriptionsModule
}

type WalletsModule struct {
	Repository *wallets.Repository
	Service    *wallets.Service
}

type ChargesModule struct {
	Repository *charges.Repository
	Service    *charges.Service
}

type LedgerModule struct {
	Repository *ledger.Repository
	Service    *ledger.Service
}

type SubscriptionsModule struct {
	Repository *subscriptions.Repository
	Service    *subscriptions.Service
}

func New(queries *sqlc.Queries) *Module {
	if queries == nil {
		panic("billing: queries are required")
	}

	walletRepository := wallets.NewRepository(
		queries,
	)
	walletService := wallets.NewService(
		walletRepository,
	)

	chargeRepository := charges.NewRepository(
		queries,
	)
	chargeService := charges.NewService(
		chargeRepository,
	)

	ledgerRepository := ledger.NewRepository(
		queries,
	)
	ledgerService := ledger.NewService(
		ledgerRepository,
	)

	subscriptionRepository := subscriptions.NewRepository(
		queries,
	)
	subscriptionService := subscriptions.NewService(
		subscriptionRepository,
	)

	return &Module{
		Wallets: WalletsModule{
			Repository: walletRepository,
			Service:    walletService,
		},
		Charges: ChargesModule{
			Repository: chargeRepository,
			Service:    chargeService,
		},
		Ledger: LedgerModule{
			Repository: ledgerRepository,
			Service:    ledgerService,
		},
		Subscriptions: SubscriptionsModule{
			Repository: subscriptionRepository,
			Service:    subscriptionService,
		},
	}
}

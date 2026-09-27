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

	walletRepository := wallets.NewRepository(queries)
	chargeRepository := charges.NewRepository(queries)
	ledgerRepository := ledger.NewRepository(queries)
	subscriptionRepository := subscriptions.NewRepository(queries)

	return &Module{
		Wallets: WalletsModule{
			Repository: walletRepository,
			Service:    wallets.NewService(walletRepository),
		},
		Charges: ChargesModule{
			Repository: chargeRepository,
			Service:    charges.NewService(chargeRepository),
		},
		Ledger: LedgerModule{
			Repository: ledgerRepository,
			Service:    ledger.NewService(ledgerRepository),
		},
		Subscriptions: SubscriptionsModule{
			Repository: subscriptionRepository,
			Service:    subscriptions.NewService(subscriptionRepository),
		},
	}
}

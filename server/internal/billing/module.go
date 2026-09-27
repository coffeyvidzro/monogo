package billing

import (
	"github.com/coffeyvidzro/monogo/internal/billing/charges"
	"github.com/coffeyvidzro/monogo/internal/billing/charging"
	"github.com/coffeyvidzro/monogo/internal/billing/ledger"
	"github.com/coffeyvidzro/monogo/internal/billing/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/billing/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
)

type Dependencies struct {
	OCS *redisintegration.OCS
}

type Module struct {
	Wallets       WalletsModule
	Charges       ChargesModule
	Charging      ChargingModule
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

type ChargingModule struct {
	Service *charging.Service
	Handler *charging.Handler
}

type LedgerModule struct {
	Repository *ledger.Repository
	Service    *ledger.Service
}

type SubscriptionsModule struct {
	Repository *subscriptions.Repository
	Service    *subscriptions.Service
}

func New(
	queries *sqlc.Queries,
	deps Dependencies,
) *Module {
	if queries == nil {
		panic("billing: queries are required")
	}
	if deps.OCS == nil {
		panic("billing: OCS is required")
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

	chargingService := charging.NewService(
		walletService,
		chargeService,
		deps.OCS,
	)
	chargingHandler := charging.NewHandler(
		chargingService,
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
		Charging: ChargingModule{
			Service: chargingService,
			Handler: chargingHandler,
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

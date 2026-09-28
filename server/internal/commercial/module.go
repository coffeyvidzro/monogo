package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/commercial/wholesale"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
)

type Module struct {
	Wallets       WalletsModule
	Subscriptions SubscriptionsModule
	Pricing       PricingModule
	Wholesale     WholesaleModule
}

type WalletsModule struct {
	Repository *wallets.Repository
	Service    *wallets.Service
}

type SubscriptionsModule struct {
	Repository *subscriptions.Repository
	Service    *subscriptions.Service
}

type PricingModule struct {
	Repository *pricing.Repository
	Service    *pricing.Service
}

type WholesaleModule struct {
	Repository *wholesale.Repository
	Service    *wholesale.Service
}

func New(queries *sqlc.Queries) *Module {
	walletsRepository := wallets.NewRepository(queries)
	walletsService := wallets.NewService(walletsRepository)

	subscriptionsRepository := subscriptions.NewRepository(queries)
	subscriptionsService := subscriptions.NewService(subscriptionsRepository)

	pricingRepository := pricing.NewRepository(queries)
	pricingService := pricing.NewService(pricingRepository)

	wholesaleRepository := wholesale.NewRepository(queries)
	wholesaleService := wholesale.NewService(wholesaleRepository)

	return &Module{
		Wallets: WalletsModule{
			Repository: walletsRepository,
			Service:    walletsService,
		},
		Subscriptions: SubscriptionsModule{
			Repository: subscriptionsRepository,
			Service:    subscriptionsService,
		},
		Pricing: PricingModule{
			Repository: pricingRepository,
			Service:    pricingService,
		},
		Wholesale: WholesaleModule{
			Repository: wholesaleRepository,
			Service:    wholesaleService,
		},
	}
}

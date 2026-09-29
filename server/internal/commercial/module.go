package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/plans"
	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/providercosts"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/commercial/wholesale"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Queries *sqlc.Queries
}

type Module struct {
	Wallets       WalletsModule
	Plans         PlansModule
	Subscriptions SubscriptionsModule
	Pricing       PricingModule
	ProviderCosts ProviderCostsModule
	Wholesale     WholesaleModule
}

type WalletsModule struct {
	Repository *wallets.Repository
	Service    *wallets.Service
	Handler    *wallets.Handler
}

type PlansModule struct {
	Repository *plans.Repository
	Service    *plans.Service
	Handler    *plans.Handler
}

type SubscriptionsModule struct {
	Repository *subscriptions.Repository
	Service    *subscriptions.Service
	Handler    *subscriptions.Handler
}

type PricingModule struct {
	Repository *pricing.Repository
	Service    *pricing.Service
	Handler    *pricing.Handler
}

type ProviderCostsModule struct {
	Service *providercosts.Service
}

type WholesaleModule struct {
	Repository *wholesale.Repository
	Service    *wholesale.Service
}

func New(deps Dependencies) *Module {
	walletsRepository := wallets.NewRepository(deps.Queries)
	walletsService := wallets.NewService(
		walletsRepository,
		deps.DB,
	)

	plansRepository := plans.NewRepository(deps.Queries)
	plansService := plans.NewService(
		plansRepository,
	)

	subscriptionsRepository := subscriptions.NewRepository(deps.Queries)
	subscriptionsService := subscriptions.NewService(
		subscriptionsRepository,
	)

	pricingRepository := pricing.NewRepository(deps.Queries)
	pricingService := pricing.NewService(
		pricingRepository,
	)

	providerCostsService := providercosts.NewService(deps.Queries)

	wholesaleRepository := wholesale.NewRepository(deps.Queries)
	wholesaleService := wholesale.NewService(
		wholesaleRepository,
	)

	return &Module{
		Wallets: WalletsModule{
			Repository: walletsRepository,
			Service:    walletsService,
			Handler:    wallets.NewHandler(walletsService),
		},
		Plans: PlansModule{
			Repository: plansRepository,
			Service:    plansService,
			Handler:    plans.NewHandler(plansService),
		},
		Subscriptions: SubscriptionsModule{
			Repository: subscriptionsRepository,
			Service:    subscriptionsService,
			Handler:    subscriptions.NewHandler(subscriptionsService),
		},
		Pricing: PricingModule{
			Repository: pricingRepository,
			Service:    pricingService,
			Handler:    pricing.NewHandler(pricingService),
		},
		ProviderCosts: ProviderCostsModule{
			Service: providerCostsService,
		},
		Wholesale: WholesaleModule{
			Repository: wholesaleRepository,
			Service:    wholesaleService,
		},
	}
}

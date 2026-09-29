package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/plans"
	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/providercharges"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Queries *sqlc.Queries
}

type Module struct {
	Wallets         WalletsModule
	Plans           PlansModule
	Subscriptions   SubscriptionsModule
	Pricing         PricingModule
	ProviderCharges ProviderChargesModule
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

type ProviderChargesModule struct {
	Repository *providercharges.Repository
	Service    *providercharges.Service
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

	providerChargesRepository := providercharges.NewRepository(deps.Queries)
	providerChargesService := providercharges.NewService(providerChargesRepository)

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
		ProviderCharges: ProviderChargesModule{
			Repository: providerChargesRepository,
			Service:    providerChargesService,
		},
	}
}

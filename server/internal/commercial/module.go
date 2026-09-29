package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/plans"
	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
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
	Payments      PaymentsModule
	Wallets       WalletsModule
	Plans         PlansModule
	Subscriptions SubscriptionsModule
	Pricing       PricingModule
}

type PaymentsModule struct {
	Repository *payments.Repository
	Service    *payments.Service
	Handler    *payments.Handler
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

	paymentsRepository := payments.NewRepository(deps.Queries)
	paymentsService := payments.NewService(
		paymentsRepository,
		subscriptionsService,
		walletsService,
	)

	pricingRepository := pricing.NewRepository(deps.Queries)
	pricingService := pricing.NewService(
		pricingRepository,
	)

	return &Module{
		Payments: PaymentsModule{
			Repository: paymentsRepository,
			Service:    paymentsService,
			Handler:    payments.NewHandler(paymentsService),
		},
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
	}
}

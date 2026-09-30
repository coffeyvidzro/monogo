package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/checkout"
	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/plans"
	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB       *pgxpool.Pool
	Queries  *sqlc.Queries
	Stripe   *stripe.Client
	Paystack *paystack.Client
}

type Module struct {
	Checkout      CheckoutModule
	Payments      PaymentsModule
	Wallets       WalletsModule
	Plans         PlansModule
	Subscriptions SubscriptionsModule
	Pricing       PricingModule
}

type CheckoutModule struct {
	Repository *checkout.Repository
	Service    *checkout.Service
	Handler    *checkout.Handler
}

type PaymentsModule struct {
	Repository *payments.Repository
	Service    *payments.Service
	Providers  *payments.ProviderService
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
	paymentsService := payments.NewService(paymentsRepository)
	paymentProviders := payments.NewProviderService(
		deps.Stripe,
		deps.Paystack,
	)

	checkoutRepository := checkout.NewRepository(deps.Queries)
	checkoutService := checkout.NewService(
		checkoutRepository,
		paymentsService,
		paymentProviders,
		subscriptionsService,
		walletsService,
		deps.DB,
	)

	pricingRepository := pricing.NewRepository(deps.Queries)
	pricingService := pricing.NewService(
		pricingRepository,
	)

	return &Module{
		Checkout: CheckoutModule{
			Repository: checkoutRepository,
			Service:    checkoutService,
			Handler:    checkout.NewHandler(checkoutService),
		},
		Payments: PaymentsModule{
			Repository: paymentsRepository,
			Service:    paymentsService,
			Providers:  paymentProviders,
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

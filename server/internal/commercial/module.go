package commercial

import (
	"github.com/coffeyvidzro/monogo/internal/commercial/checkout"
	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/plans"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Queries *sqlc.Queries
	Stripe  *stripe.Client
}

type Module struct {
	Checkout      CheckoutModule
	Payments      PaymentsModule
	Plans         PlansModule
	Subscriptions SubscriptionsModule
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

func New(deps Dependencies) *Module {
	plansRepository := plans.NewRepository(deps.Queries)
	plansService := plans.NewService(plansRepository)

	subscriptionsRepository := subscriptions.NewRepository(deps.Queries)
	subscriptionsService := subscriptions.NewService(subscriptionsRepository)

	paymentsRepository := payments.NewRepository(deps.Queries)
	paymentsService := payments.NewService(paymentsRepository)
	paymentProviders := payments.NewProviderService(deps.Stripe)

	checkoutRepository := checkout.NewRepository(deps.Queries)
	checkoutService := checkout.NewService(
		checkoutRepository,
		paymentsService,
		paymentProviders,
		subscriptionsService,
		deps.DB,
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
	}
}

package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/go-chi/chi/v5"
)

const paymentWebhookBodyLimit int64 = 1 << 20

func registerPaymentProviderRoutes(
	router chi.Router,
	modules *modules,
) {
	router.Post(
		"/v1/provider-webhooks/payments/stripe",
		func(w http.ResponseWriter, r *http.Request) {
			payload, err := readPaymentWebhookBody(r)
			if err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}

			webhook, err := modules.commercial.Payments.Providers.ParseStripeWebhook(
				payload,
				r.Header.Get("Stripe-Signature"),
				time.Now().UTC(),
			)
			if err != nil {
				http.Error(w, "invalid Stripe webhook", http.StatusUnauthorized)
				return
			}

			if err := processPaymentProviderWebhook(
				r.Context(),
				modules,
				webhook,
				payload,
			); err != nil {
				http.Error(w, "process Stripe webhook", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		},
	)

	router.Post(
		"/v1/provider-webhooks/payments/paystack",
		func(w http.ResponseWriter, r *http.Request) {
			payload, err := readPaymentWebhookBody(r)
			if err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}

			webhook, err := modules.commercial.Payments.Providers.ParsePaystackWebhook(
				payload,
				r.Header.Get("X-Paystack-Signature"),
			)
			if err != nil {
				http.Error(w, "invalid Paystack webhook", http.StatusUnauthorized)
				return
			}

			if err := processPaymentProviderWebhook(
				r.Context(),
				modules,
				webhook,
				payload,
			); err != nil {
				http.Error(w, "process Paystack webhook", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		},
	)
}

func readPaymentWebhookBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, paymentWebhookBodyLimit))
}

func processPaymentProviderWebhook(
	ctx context.Context,
	modules *modules,
	webhook payments.ProviderWebhook,
	payload []byte,
) error {
	event, payment, err := modules.commercial.Payments.Service.RecordProviderEvent(
		ctx,
		payments.RecordProviderEventRequest{
			Provider:          webhook.Provider,
			ProviderPaymentID: webhook.ProviderPaymentID,
			ProviderEventID:   webhook.ProviderEventID,
			EventType:         webhook.EventType,
			Payload:           payload,
			ReceivedAt:        time.Now().UTC(),
		},
	)
	if err != nil {
		return err
	}
	if event.ProcessedAt != nil {
		return nil
	}

	if payments.IsSuccessfulProviderEvent(
		webhook.Provider,
		webhook.EventType,
	) {
		paidAt, err := modules.commercial.Payments.Providers.Verify(
			ctx,
			payment,
		)
		if err != nil {
			return err
		}

		payment, err = modules.commercial.Payments.Service.MarkSucceeded(
			ctx,
			payment,
			paidAt,
		)
		if err != nil {
			return err
		}

		if _, err := modules.commercial.Checkout.Service.Complete(
			ctx,
			payment.OrganizationID,
			payment.CheckoutID,
			paidAt,
		); err != nil {
			return err
		}
	}

	_, err = modules.commercial.Payments.Service.MarkProviderEventProcessed(
		ctx,
		event.ID,
		time.Now().UTC(),
	)
	return err
}

package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
	domain "github.com/coffeyvidzro/monogo/internal/telecom/messaging"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type whatsappConnectionConfiguration struct {
	BaseURL       string `json:"base_url"`
	PhoneNumberID string `json:"phone_number_id"`
}
type whatsappConnectionSecrets struct {
	AccessToken string `json:"access_token"`
	AppSecret   string `json:"app_secret"`
	VerifyToken string `json:"verify_token"`
}

func registerMessagingProviderRoutes(router chi.Router, modules *modules) {
	router.Get("/v1/provider-webhooks/whatsapp/{connectionID}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("hub.mode") != "subscribe" {
			http.Error(w, "invalid verification mode", http.StatusBadRequest)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "connectionID"))
		if err != nil {
			http.Error(w, "invalid connection", http.StatusBadRequest)
			return
		}
		connection, err := modules.queries.GetMessagingConnection(r.Context(), id)
		if err != nil || connection.Channel != "whatsapp" {
			http.NotFound(w, r)
			return
		}
		secret, err := modules.credentialCipher.Decrypt(connection.EncryptedSecret)
		if err != nil {
			http.Error(w, "invalid connection credentials", http.StatusInternalServerError)
			return
		}
		var secrets whatsappConnectionSecrets
		if err := json.Unmarshal([]byte(secret), &secrets); err != nil {
			http.Error(w, "invalid connection credentials", http.StatusInternalServerError)
			return
		}
		if secrets.VerifyToken == "" || r.URL.Query().Get("hub.verify_token") != secrets.VerifyToken {
			http.Error(w, "invalid verification token", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(r.URL.Query().Get("hub.challenge")))
	})
	router.Post("/v1/provider-webhooks/whatsapp/{connectionID}", func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "connectionID"))
		if err != nil {
			http.Error(w, "invalid connection", http.StatusBadRequest)
			return
		}
		connection, err := modules.queries.GetMessagingConnection(r.Context(), id)
		if err != nil || connection.Channel != "whatsapp" {
			http.NotFound(w, r)
			return
		}
		secret, err := modules.credentialCipher.Decrypt(connection.EncryptedSecret)
		if err != nil {
			http.Error(w, "invalid connection credentials", http.StatusInternalServerError)
			return
		}
		var config whatsappConnectionConfiguration
		var secrets whatsappConnectionSecrets
		if json.Unmarshal(connection.Configuration, &config) != nil || json.Unmarshal([]byte(secret), &secrets) != nil {
			http.Error(w, "invalid connection configuration", http.StatusInternalServerError)
			return
		}
		client, err := whatsapp.New(whatsapp.Config{
			BaseURL:       config.BaseURL,
			PhoneNumberID: config.PhoneNumberID,
			AccessToken:   secrets.AccessToken,
			AppSecret:     secrets.AppSecret,
			Timeout:       10 * time.Second,
		}, nil)
		if err != nil {
			http.Error(w, "invalid connection configuration", http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if !client.VerifySignature(body, r.Header.Get("X-Hub-Signature-256")) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		events, err := whatsapp.ParseWebhook(body)
		if err != nil {
			http.Error(w, "invalid webhook", http.StatusBadRequest)
			return
		}
		for _, event := range events {
			if event.PhoneNumberID == "" || event.PhoneNumberID != config.PhoneNumberID {
				http.Error(w, "webhook phone number does not match connection", http.StatusUnprocessableEntity)
				return
			}
			switch event.Kind {
			case whatsapp.EventMessage:
				organizationID, resolveErr := modules.queries.ResolveInboundMessagingOrganization(
					r.Context(),
					connection.ID,
				)
				if resolveErr != nil {
					http.Error(w, "resolve inbound organization", http.StatusUnprocessableEntity)
					return
				}
				text := event.Text
				providerMessageID := event.MessageID
				if _, err = modules.telecom.Messaging.Service.AcceptInbound(r.Context(), domain.InboundRequest{
					OrganizationID:        organizationID,
					MessagingConnectionID: connection.ID,
					ProviderMessageID:     &providerMessageID,
					Channel:               domain.ChannelWhatsApp,
					From:                  event.From,
					To:                    event.To,
					Body:                  &text,
				}); err != nil {
					http.Error(w, "accept message", http.StatusInternalServerError)
					return
				}
			case whatsapp.EventStatus:
				status := domain.DeliveryStatus(event.Status)
				if status == "read" {
					status = domain.DeliveryDelivered
				}
				code, message := event.ErrorCode, event.ErrorMessage
				if _, err = modules.telecom.Messaging.Service.AcceptDelivery(r.Context(), domain.DeliveryReceipt{
					MessagingConnectionID: connection.ID,
					ProviderMessageID:     event.MessageID,
					Status:                status,
					FailureCode:           &code,
					FailureMessage:        &message,
				}); err != nil {
					http.Error(w, "accept status", http.StatusInternalServerError)
					return
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

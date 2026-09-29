package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
	runtimemessaging "github.com/coffeyvidzro/monogo/internal/runtime/messaging"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	domain "github.com/coffeyvidzro/monogo/internal/telecom/messaging"
	"github.com/jackc/pgx/v5/pgxpool"
	natsjs "github.com/nats-io/nats.go/jetstream"
)

type messagingRuntime struct {
	nats        *natsintegration.Client
	consumer    *domain.Consumer
	jobs        *domain.Jobs
	connections []sqlc.MessagingConnection
	smpp        map[string]*smpp.Client
}
type smsConnectionConfig struct {
	Host       string        `json:"host"`
	Port       int           `json:"port"`
	SystemType string        `json:"system_type"`
	BindMode   smpp.BindMode `json:"bind_mode"`
}
type smsConnectionSecrets struct {
	SystemID string `json:"system_id"`
	Password string `json:"password"`
}
type whatsappConnectionConfig struct {
	BaseURL       string `json:"base_url"`
	PhoneNumberID string `json:"phone_number_id"`
}
type whatsappSecrets struct {
	AccessToken string `json:"access_token"`
	AppSecret   string `json:"app_secret"`
	VerifyToken string `json:"verify_token"`
}

func newMessagingRuntime(ctx context.Context, queries *sqlc.Queries, db *pgxpool.Pool, nats *natsintegration.Client, cipher *encryption.Cipher) (*messagingRuntime, error) {
	connections, err := queries.ListActiveMessagingConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list messaging connections: %w", err)
	}
	controller := runtimemessaging.NewController()
	clients := make(map[string]*smpp.Client)
	initialized := false
	defer func() {
		if initialized {
			return
		}
		for _, client := range clients {
			_ = client.Close()
		}
	}()
	for _, connection := range connections {
		secret, err := cipher.Decrypt(connection.EncryptedSecret)
		if err != nil {
			return nil, fmt.Errorf("decrypt messaging connection %s: %w", connection.ID, err)
		}
		switch connection.Channel {
		case "sms":
			var value smsConnectionConfig
			var secrets smsConnectionSecrets
			if err := json.Unmarshal(connection.Configuration, &value); err != nil {
				return nil, err
			}
			if err := json.Unmarshal([]byte(secret), &secrets); err != nil {
				return nil, fmt.Errorf("decode SMPP credentials for connection %s: %w", connection.ID, err)
			}
			config := smpp.DefaultConfig(value.Host, value.Port, secrets.SystemID, secrets.Password)
			config.SystemType = value.SystemType
			config.BindMode = value.BindMode
			client, err := smpp.New(config)
			if err != nil {
				return nil, err
			}
			if err := client.Start(ctx); err != nil {
				return nil, err
			}
			controller.RegisterSMS(connection.ID, client)
			clients[connection.ID.String()] = client
		case "whatsapp":
			var value whatsappConnectionConfig
			var secrets whatsappSecrets
			if err := json.Unmarshal(connection.Configuration, &value); err != nil {
				return nil, err
			}
			if err := json.Unmarshal([]byte(secret), &secrets); err != nil {
				return nil, err
			}
			client, err := whatsapp.New(whatsapp.Config{
				BaseURL:       value.BaseURL,
				PhoneNumberID: value.PhoneNumberID,
				AccessToken:   secrets.AccessToken,
				AppSecret:     secrets.AppSecret,
				Timeout:       10 * time.Second,
			}, nil)
			if err != nil {
				return nil, err
			}
			controller.RegisterWhatsApp(connection.ID, client)
		}
	}
	repo := domain.NewRepository(db)
	service := domain.NewService(repo)
	pricingService := pricing.NewService(
		pricing.NewRepository(queries),
	)
	walletService := wallets.NewService(
		wallets.NewRepository(queries),
		db,
	)
	service.ConfigureBilling(
		pricingService,
		walletService,
	)
	initialized = true
	return &messagingRuntime{
		nats:        nats,
		consumer:    domain.NewConsumer(service, repo, controller),
		jobs:        domain.NewJobs(service, repo),
		connections: connections,
		smpp:        clients,
	}, nil
}
func (m *messagingRuntime) RunOutbound(ctx context.Context) error {
	consumer, err := m.nats.CreateOrUpdateConsumer(
		ctx,
		natsintegration.EventsStreamName,
		natsjs.ConsumerConfig{
			Name:          "messaging-outbound",
			Durable:       "messaging-outbound",
			FilterSubject: natsintegration.EventsSubjectPrefix + string(domain.EventQueued),
			AckPolicy:     natsjs.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    20,
		},
	)
	if err != nil {
		return err
	}
	err = m.nats.Consume(ctx, consumer, func(messageCtx context.Context, message natsjs.Msg) (natsintegration.AckAction, error) {
		var event domain.Event
		if err := json.Unmarshal(message.Data(), &event); err != nil {
			return natsintegration.Term, err
		}
		value, err := m.consumer.HandleQueuedByID(
			messageCtx,
			event.OrganizationID,
			event.MessageID,
		)
		if err != nil {
			return natsintegration.Nak, err
		}
		_ = value
		return natsintegration.Ack, nil
	})
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
func (m *messagingRuntime) RunInbound(ctx context.Context) error {
	for _, connection := range m.connections {
		client := m.smpp[connection.ID.String()]
		if client != nil {
			go m.superviseSMPP(ctx, connection, client)
		}
	}
	<-ctx.Done()
	return nil
}

func (m *messagingRuntime) superviseSMPP(
	ctx context.Context,
	connection sqlc.MessagingConnection,
	client *smpp.Client,
) {
	for ctx.Err() == nil {
		if err := m.jobs.RunSMPP(ctx, connection, client); err == nil {
			return
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (m *messagingRuntime) Close() error {
	var result error
	for id, client := range m.smpp {
		if err := client.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close SMPP connection %s: %w", id, err))
		}
	}
	return result
}

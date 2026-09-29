package messaging

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	runtimemessaging "github.com/coffeyvidzro/monogo/internal/runtime/messaging"
	"github.com/google/uuid"
)

// Consumer owns canonical state transitions and delegates execution to the runtime.
type Consumer struct {
	service *Service
	repo    *Repository
	runtime *runtimemessaging.Controller
}

func NewConsumer(service *Service, repo *Repository, runtime *runtimemessaging.Controller) *Consumer {
	if service == nil || repo == nil || runtime == nil {
		panic("messaging: service, repository, and runtime are required")
	}
	return &Consumer{
		service: service,
		repo:    repo,
		runtime: runtime,
	}
}

// HandleQueuedByID reloads current durable state. Outbox payloads are
// historical snapshots and must never drive an external side effect directly.
func (c *Consumer) HandleQueuedByID(
	ctx context.Context,
	organizationID, messageID uuid.UUID,
) (sqlc.Message, error) {
	message, err := c.service.Get(ctx, organizationID, messageID)
	if err != nil {
		return sqlc.Message{}, err
	}
	if message.Status != string(StatusQueued) {
		return message, nil
	}
	return c.HandleQueued(ctx, message)
}

func (c *Consumer) HandleQueued(ctx context.Context, message sqlc.Message) (sqlc.Message, error) {
	if message.Direction != string(DirectionOutbound) || message.Status != string(StatusQueued) {
		return sqlc.Message{}, fmt.Errorf("message %s is not queued outbound", message.ID)
	}
	connection, err := c.repo.ResolveConnection(ctx, message.OrganizationID, Channel(message.Channel))
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("resolve messaging connection: %w", err)
	}
	managed := connection.Scope == "platform"
	_, err = c.service.SetProviderAttribution(ctx, message.OrganizationID, message.ID, connection.ID)
	if err != nil {
		return sqlc.Message{}, err
	}
	submitting, err := c.service.BeginSubmission(ctx, message.OrganizationID, message.ID)
	if err != nil {
		return sqlc.Message{}, err
	}
	request, err := runtimemessaging.Translate(
		submitting.ID,
		connection.ID,
		runtimemessaging.Channel(submitting.Channel),
		submitting.FromAddress,
		submitting.ToAddress,
		submitting.Body,
	)
	if err != nil {
		return sqlc.Message{}, err
	}
	if managed {
		if err := c.service.AuthorizeManagedOutbound(ctx, message); err != nil {
			if _, requeueErr := c.service.RequeueSubmission(
				ctx,
				message.OrganizationID,
				message.ID,
			); requeueErr != nil {
				return sqlc.Message{}, errors.Join(err, requeueErr)
			}

			return sqlc.Message{}, err
		}
	}
	result, err := c.runtime.Send(ctx, request)
	if err != nil {
		var submissionError *runtimemessaging.SubmissionError
		if errors.As(err, &submissionError) {
			switch submissionError.Outcome {
			case runtimemessaging.SubmissionNotSubmitted:
				if managed {
					_ = c.service.ReleaseManagedOutbound(ctx, message)
				}
				if _, requeueErr := c.service.RequeueSubmission(ctx, message.OrganizationID, message.ID); requeueErr != nil {
					return sqlc.Message{}, errors.Join(err, requeueErr)
				}
				return sqlc.Message{}, fmt.Errorf("submit %s message: %w", message.Channel, err)
			case runtimemessaging.SubmissionRejected:
				if managed {
					_ = c.service.ReleaseManagedOutbound(ctx, message)
				}
				code := "provider_rejected"
				detail := err.Error()
				return c.service.MarkFailed(ctx, message.OrganizationID, message.ID, Failure{
					Code:    &code,
					Message: &detail,
				})
			}
		}
		code := "submission_outcome_unknown"
		detail := fmt.Sprintf("%s submission may have reached the provider: %v", message.Channel, err)
		if managed {
			if captureErr := c.service.CaptureManagedOutbound(ctx, message); captureErr != nil {
				return sqlc.Message{}, errors.Join(err, captureErr)
			}
		}
		return c.service.MarkSubmissionUnknown(ctx, message.OrganizationID, message.ID, Failure{
			Code:    &code,
			Message: &detail,
		})
	}
	providerID := strings.TrimSpace(result.ExternalID)
	if providerID == "" {
		code := "submission_outcome_unknown"
		detail := "transport accepted message without returning a provider message id"
		if managed {
			if captureErr := c.service.CaptureManagedOutbound(ctx, message); captureErr != nil {
				return sqlc.Message{}, captureErr
			}
		}
		return c.service.MarkSubmissionUnknown(ctx, message.OrganizationID, message.ID, Failure{
			Code:    &code,
			Message: &detail,
		})
	}
	if managed {
		if err := c.service.CaptureManagedOutbound(ctx, message); err != nil {
			return sqlc.Message{}, err
		}
	}
	return c.service.MarkSubmitted(ctx, message.OrganizationID, message.ID, providerID)
}

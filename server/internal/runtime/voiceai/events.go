package voiceai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	natsjs "github.com/nats-io/nats.go/jetstream"
)

func (r *Runtime) ConfigureEvents(
	db *pgxpool.Pool,
	natsClient *natsintegration.Client,
) {
	r.db = db
	r.nats = natsClient
}

func (r *Runtime) RunEvents(ctx context.Context) error {
	if r.db == nil || r.nats == nil {
		return fmt.Errorf("voice AI event dependencies are required")
	}
	consumer, err := r.nats.CreateOrUpdateConsumer(
		ctx,
		natsintegration.EventsStreamName,
		natsjs.ConsumerConfig{
			Name:          "voice-agent-media",
			Durable:       "voice-agent-media",
			FilterSubject: natsintegration.VoiceAgentMediaEventSubject,
			AckPolicy:     natsjs.AckExplicitPolicy,
			AckWait:       45 * time.Second,
			MaxDeliver:    20,
		},
	)
	if err != nil {
		return err
	}
	err = r.nats.Consume(
		ctx,
		consumer,
		func(messageCtx context.Context, message natsjs.Msg) (natsintegration.AckAction, error) {
			var event session.PublishedEvent
			if err := json.Unmarshal(message.Data(), &event); err != nil {
				return natsintegration.Term, err
			}
			if err := r.handleMediaEvent(messageCtx, event); err != nil {
				return natsintegration.Nak, err
			}
			return natsintegration.Ack, nil
		},
	)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (r *Runtime) handleMediaEvent(
	ctx context.Context,
	event session.PublishedEvent,
) error {
	processed, err := r.recordMediaEvent(ctx, event)
	if err != nil || processed {
		return err
	}
	switch event.Type {
	case session.EventTranscriptFinal:
		err = r.persistTurn(ctx, event, "user", event.Text, "", "")
	case session.EventResponseStopped:
		if strings.TrimSpace(event.Text) != "" {
			err = r.persistTurn(ctx, event, "assistant", event.Text, "", "")
		}
	case session.EventToolCall:
		err = r.executeTool(ctx, event)
	case session.EventInterrupted:
		_, err = r.db.Exec(
			ctx,
			`UPDATE voice_agent_sessions
             SET interruption_count = interruption_count + 1
             WHERE id = $1 AND organization_id = $2`,
			event.SessionID,
			event.OrganizationID,
		)
	case session.EventUsage, session.EventError, session.EventSpeechStarted:
	default:
		err = fmt.Errorf("unsupported media event type %q", event.Type)
	}
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		ctx,
		`UPDATE voice_agent_media_events
         SET processed_at = now()
         WHERE session_id = $1 AND generation = $2`,
		event.SessionID,
		int64(event.Generation),
	)
	return err
}

func (r *Runtime) recordMediaEvent(
	ctx context.Context,
	event session.PublishedEvent,
) (bool, error) {
	payload := event.Payload
	if len(payload) == 0 || !json.Valid(payload) || payload[0] != '{' {
		payload, _ = json.Marshal(map[string]string{
			"raw": string(event.Payload),
		})
	}
	var processedAt *time.Time
	err := r.db.QueryRow(
		ctx,
		`INSERT INTO voice_agent_media_events (
             organization_id, session_id, generation, event_type, provider_id,
             tool_name, text_content, payload, occurred_at
         )
         SELECT $1, session.id, $5, $6, NULLIF($7, ''), NULLIF($8, ''),
                NULLIF($9, ''), $10, $11
         FROM voice_agent_sessions AS session
         WHERE session.id = $2
           AND session.organization_id = $1
           AND session.call_id = $3
           AND session.voice_agent_id = $4
		 ON CONFLICT DO NOTHING
		 RETURNING processed_at`,
		event.OrganizationID,
		event.SessionID,
		event.CallID,
		event.VoiceAgentID,
		int64(event.Generation),
		string(event.Type),
		event.ProviderID,
		event.ToolName,
		event.Text,
		payload,
		event.OccurredAt,
	).Scan(&processedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("record Voice Agent media event: %w", err)
	}
	return processedAt != nil, nil
}

func (r *Runtime) persistTurn(
	ctx context.Context,
	event session.PublishedEvent,
	role string,
	content string,
	toolName string,
	toolCallID string,
) error {
	if event.Generation > uint64(^uint32(0)>>1) {
		return fmt.Errorf("media event generation exceeds turn sequence range")
	}
	providerID := strings.TrimSpace(event.ProviderID)
	_, err := r.orchestrator.CreateTurn(ctx, conversations.Identity{
		OrganizationID: event.OrganizationID,
		SessionID:      event.SessionID,
	}, conversations.CreateTurnRequest{
		Sequence:   int32(event.Generation),
		Role:       role,
		Content:    content,
		ProviderID: stringPointer(providerID),
		ToolName:   stringPointer(toolName),
		ToolCallID: stringPointer(toolCallID),
		Metadata:   json.RawMessage(`{}`),
	})
	if isConflict(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		ctx,
		`UPDATE voice_agent_sessions
         SET turn_count = turn_count + 1
         WHERE id = $1 AND organization_id = $2`,
		event.SessionID,
		event.OrganizationID,
	)
	return err
}

func (r *Runtime) executeTool(ctx context.Context, event session.PublishedEvent) error {
	result, executeErr := r.orchestrator.ExecuteNamedTool(ctx, tools.ExecuteNamedRequest{
		OrganizationID: event.OrganizationID,
		VoiceAgentID:   event.VoiceAgentID,
		SessionID:      event.SessionID,
		CallID:         event.CallID,
		ToolName:       event.ToolName,
		ToolCallID:     event.ProviderID,
		Arguments:      json.RawMessage(event.Text),
	})
	content := result.Body
	if executeErr != nil {
		content = []byte(executeErr.Error())
	}
	deliverErr := r.media.DeliverToolResult(ctx, event.SessionID, session.ToolResult{
		Generation: event.Generation,
		CallID:     event.ProviderID,
		Name:       event.ToolName,
		Content:    content,
		IsError:    executeErr != nil,
	})
	if deliverErr != nil {
		return deliverErr
	}
	turnErr := r.persistTurn(
		ctx,
		event,
		"tool",
		string(content),
		event.ToolName,
		event.ProviderID,
	)
	return turnErr
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func isConflict(err error) bool {
	var appErr *apperror.AppError
	return errors.As(err, &appErr) && appErr.Code == "CONFLICT"
}

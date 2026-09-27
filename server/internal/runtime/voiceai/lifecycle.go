package voiceai

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/ai/orchestration"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/google/uuid"
)

const voiceAgentSessionVariable = "leamout_voice_agent_session_id"

func (r *Runtime) HandleLifecycle(
	ctx context.Context,
	call sqlc.Call,
	event calling.LifecycleEvent,
) error {
	switch event.Type {
	case calling.LifecycleAnswered:
		return r.attach(ctx, call, event.ChannelID)
	case calling.LifecycleCompleted:
		return r.finish(ctx, call, "completed", event.OccurredAt)
	case calling.LifecycleFailed:
		return r.finish(ctx, call, "failed", event.OccurredAt)
	case calling.LifecycleCancelled:
		return r.finish(ctx, call, "cancelled", event.OccurredAt)
	default:
		return nil
	}
}

func (r *Runtime) attach(ctx context.Context, call sqlc.Call, channelID string) error {
	if call.ApplicationID == nil {
		return nil
	}
	channelUUID, err := uuid.Parse(strings.TrimSpace(channelID))
	if err != nil {
		return fmt.Errorf("voice AI channel id is invalid: %w", err)
	}

	record, attached, err := r.orchestrator.AttachCall(
		ctx,
		call.OrganizationID,
		call.ID,
		*call.ApplicationID,
	)
	if err != nil {
		return fmt.Errorf("attach durable Voice Agent session: %w", err)
	}
	if !attached {
		return nil
	}

	existing, err := r.freeSwitch.GetVariable(ctx, channelID, voiceAgentSessionVariable)
	if err != nil {
		return fmt.Errorf("read Voice Agent channel attachment: %w", err)
	}
	existing = strings.TrimSpace(existing)
	if existing != "" && existing != "_undef_" {
		if existing == record.ID.String() {
			return nil
		}
		return fmt.Errorf("FreeSWITCH channel is already attached to Voice Agent session %s", existing)
	}

	format, err := mediaFormat(record.Engine)
	if err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return err
	}
	cfg := orchestration.MediaConfigFromSession(record, session.Config{
		ID:             record.ID,
		OrganizationID: call.OrganizationID,
		CallID:         call.ID,
		ChannelID:      channelUUID,
		InputFormat:    format,
		OutputFormat:   format,
	})

	websocketURL, err := r.media.CreateSession(ctx, cfg)
	if err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return fmt.Errorf("create Voice Agent media session: %w", err)
	}

	reply, err := r.freeSwitch.StartAudioForkWithReply(ctx, freeswitch.AudioForkRequest{
		ChannelID:    channelID,
		WebSocketURL: websocketURL,
		MixType:      "mono",
		SampleRateHz: format.SampleRateHz,
	})
	if err != nil {
		stopErr := r.media.StopSession(ctx, record.ID)
		failErr := r.failSession(ctx, call, time.Now().UTC())
		return errors.Join(fmt.Errorf("start Voice Agent audio fork: %w", err), stopErr, failErr)
	}
	if r.logger != nil {
		r.logger.Info(
			ctx,
			"Voice Agent audio fork start accepted",
			"call_id", call.ID,
			"channel_id", channelID,
			"voice_agent_session_id", record.ID,
			"sample_rate_hz", format.SampleRateHz,
			"websocket_url", redactWebSocketURL(websocketURL),
			"freeswitch_command", fmt.Sprintf(
				"uuid_audio_fork %s start %s mono %d <metadata>",
				channelID,
				redactWebSocketURL(websocketURL),
				format.SampleRateHz,
			),
			"freeswitch_reply_text", strings.TrimSpace(reply.Text),
			"freeswitch_reply_body", strings.TrimSpace(reply.Body),
		)
	}

	if err := r.freeSwitch.SetVariable(ctx, channelID, voiceAgentSessionVariable, record.ID.String()); err != nil {
		forkErr := r.freeSwitch.StopAudioFork(ctx, channelID)
		stopErr := r.media.StopSession(ctx, record.ID)
		failErr := r.failSession(ctx, call, time.Now().UTC())
		return errors.Join(
			fmt.Errorf("persist Voice Agent channel attachment: %w", err),
			forkErr,
			stopErr,
			failErr,
		)
	}
	return nil
}

func (r *Runtime) finish(
	ctx context.Context,
	call sqlc.Call,
	state string,
	endedAt time.Time,
) error {
	record, completed, err := r.orchestrator.CompleteCall(
		ctx,
		call.OrganizationID,
		call.ID,
		state,
		endedAt,
	)
	if err != nil {
		return fmt.Errorf("complete durable Voice Agent session: %w", err)
	}
	if !completed {
		return nil
	}
	if err := r.media.StopSession(ctx, record.ID); err != nil {
		return fmt.Errorf("stop Voice Agent media session: %w", err)
	}
	return nil
}

func (r *Runtime) failSession(ctx context.Context, call sqlc.Call, endedAt time.Time) error {
	_, _, err := r.orchestrator.CompleteCall(
		ctx,
		call.OrganizationID,
		call.ID,
		"failed",
		endedAt,
	)
	return err
}

func mediaFormat(engine string) (session.AudioFormat, error) {
	switch session.Engine(engine) {
	case session.EngineComposable:
		return session.AudioFormat{SampleRateHz: 16000, Channels: 1}, nil
	case session.EngineIntegrated:
		return session.AudioFormat{SampleRateHz: 24000, Channels: 1}, nil
	default:
		return session.AudioFormat{}, fmt.Errorf("unsupported Voice Agent engine %q", engine)
	}
}

func redactWebSocketURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "<invalid>"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

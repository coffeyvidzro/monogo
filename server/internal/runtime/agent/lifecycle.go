package agent

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
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
	if call.VoiceAgentID == nil {
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
		*call.VoiceAgentID,
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

	profile, err := session.ProfileForEngine(session.Engine(record.Engine))
	if err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return err
	}
	cfg := orchestration.MediaConfigFromSession(record, session.Config{
		ID:             record.ID,
		OrganizationID: call.OrganizationID,
		CallID:         call.ID,
		ChannelID:      channelUUID,
		InputFormat:    profile.InputFormat,
		OutputFormat:   profile.OutputFormat,
	})
	toolDefinitions, err := r.orchestrator.ToolDefinitions(
		ctx,
		call.OrganizationID,
		*call.VoiceAgentID,
	)
	if err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return fmt.Errorf("resolve Voice Agent tools: %w", err)
	}
	cfg.Tools = toolDefinitions
	if err := r.orchestrator.ValidateProviderTopology(
		ctx,
		call.OrganizationID,
		*call.VoiceAgentID,
		session.Engine(record.Engine),
	); err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return fmt.Errorf("validate Voice Agent provider topology: %w", err)
	}
	providerRuntimes, err := r.orchestrator.ProviderRuntimes(
		ctx,
		call.OrganizationID,
		*call.VoiceAgentID,
	)
	if err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return fmt.Errorf("resolve Voice Agent providers: %w", err)
	}
	cfg.Providers = providerRuntimes

	endpoints, err := r.media.CreateSession(ctx, cfg)
	if err != nil {
		_ = r.failSession(ctx, call, time.Now().UTC())
		return fmt.Errorf("create Voice Agent media session: %w", err)
	}
	if err := r.registerControl(ctx, call, record.ID, endpoints.ControlURL); err != nil {
		stopErr := r.stopMediaSession(ctx, record.ID)
		failErr := r.failSession(ctx, call, time.Now().UTC())
		return errors.Join(fmt.Errorf("attach Voice Agent media control: %w", err), stopErr, failErr)
	}

	reply, err := r.freeSwitch.StartAudioForkWithReply(ctx, freeswitch.AudioForkRequest{
		ChannelID:    channelID,
		WebSocketURL: endpoints.AudioURL,
		MixType:      "mono",
		SampleRateHz: profile.InputFormat.SampleRateHz,
	})
	if err != nil {
		stopErr := r.stopMediaSession(ctx, record.ID)
		failErr := r.failSession(ctx, call, time.Now().UTC())
		return errors.Join(fmt.Errorf("start Voice Agent audio fork: %w", err), stopErr, failErr)
	}
	if err := r.freeSwitch.StartAudioClock(ctx, channelID); err != nil {
		forkErr := r.freeSwitch.StopAudioFork(ctx, channelID)
		clockErr := r.freeSwitch.Break(ctx, channelID)
		stopErr := r.stopMediaSession(ctx, record.ID)
		failErr := r.failSession(ctx, call, time.Now().UTC())
		return errors.Join(
			fmt.Errorf("start Voice Agent audio clock: %w", err),
			forkErr,
			clockErr,
			stopErr,
			failErr,
		)
	}
	if r.logger != nil {
		r.logger.Info(
			ctx,
			"Voice Agent audio fork start accepted",
			"call_id", call.ID,
			"channel_id", channelID,
			"voice_agent_session_id", record.ID,
			"sample_rate_hz", profile.InputFormat.SampleRateHz,
			"websocket_url", redactWebSocketURL(endpoints.AudioURL),
			"freeswitch_command", fmt.Sprintf(
				"uuid_audio_fork %s start %s mono %d",
				channelID,
				redactWebSocketURL(endpoints.AudioURL),
				profile.InputFormat.SampleRateHz,
			),
			"freeswitch_reply_text", strings.TrimSpace(reply.Text),
			"freeswitch_reply_body", strings.TrimSpace(reply.Body),
		)
	}

	if err := r.freeSwitch.SetVariable(ctx, channelID, voiceAgentSessionVariable, record.ID.String()); err != nil {
		forkErr := r.freeSwitch.StopAudioFork(ctx, channelID)
		stopErr := r.stopMediaSession(ctx, record.ID)
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
	summary, sessionID := r.conversationSummary(call.ID)
	record, completed, err := r.orchestrator.CompleteCallWithSummary(
		ctx,
		call.OrganizationID,
		call.ID,
		state,
		endedAt,
		conversations.CompleteRequest{
			TurnCount:              summary.TurnCount,
			InterruptionCount:      summary.InterruptionCount,
			FirstResponseLatencyMS: summary.FirstResponseLatencyMS,
			AverageTurnLatencyMS:   summary.AverageTurnLatencyMS,
		},
	)
	if err != nil {
		return fmt.Errorf("complete durable Voice Agent session: %w", err)
	}
	if !completed {
		return nil
	}
	r.releaseConversation(call.ID, sessionID)
	if err := r.stopMediaSession(ctx, record.ID); err != nil {
		return fmt.Errorf("stop Voice Agent media session: %w", err)
	}
	return nil
}

func (r *Runtime) failSession(ctx context.Context, call sqlc.Call, endedAt time.Time) error {
	summary, sessionID := r.conversationSummary(call.ID)
	_, _, err := r.orchestrator.CompleteCallWithSummary(
		ctx,
		call.OrganizationID,
		call.ID,
		"failed",
		endedAt,
		conversations.CompleteRequest{
			TurnCount:              summary.TurnCount,
			InterruptionCount:      summary.InterruptionCount,
			FirstResponseLatencyMS: summary.FirstResponseLatencyMS,
			AverageTurnLatencyMS:   summary.AverageTurnLatencyMS,
		},
	)
	if err == nil {
		r.releaseConversation(call.ID, sessionID)
	}
	return err
}

func (r *Runtime) conversationSummary(callID uuid.UUID) (conversationSummary, uuid.UUID) {
	r.mu.Lock()
	sessionID := r.callSessions[callID]
	state := r.states[sessionID]
	r.mu.Unlock()
	if state == nil {
		return conversationSummary{}, sessionID
	}
	return state.summary(), sessionID
}

func (r *Runtime) releaseConversation(callID, sessionID uuid.UUID) {
	r.mu.Lock()
	delete(r.callSessions, callID)
	delete(r.states, sessionID)
	r.mu.Unlock()
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

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

type controlHTTPError struct {
	status int
	err    error
}

func (e *controlHTTPError) Error() string {
	return fmt.Sprintf("connect media control: HTTP %d: %v", e.status, e.err)
}

func (e *controlHTTPError) Unwrap() error { return e.err }

type mediaControl struct {
	connection *websocket.Conn
	events     chan session.Event
	cancel     context.CancelFunc
	closeOnce  sync.Once
	writeMu    sync.Mutex
}

func (c *mediaClient) OpenControl(ctx context.Context, rawURL string) (*mediaControl, error) {
	if ctx == nil {
		return nil, fmt.Errorf("media control context is required")
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.token)
	connection, response, err := websocket.Dial(ctx, rawURL, &websocket.DialOptions{
		HTTPHeader:      header,
		CompressionMode: websocket.CompressionDisabled,
	})
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if err != nil {
		if response != nil {
			return nil, &controlHTTPError{status: response.StatusCode, err: err}
		}
		return nil, fmt.Errorf("connect media control: %w", err)
	}
	controlCtx, cancel := context.WithCancel(context.Background())
	control := &mediaControl{
		connection: connection,
		events:     make(chan session.Event, 64),
		cancel:     cancel,
	}
	go control.readLoop(controlCtx)
	return control, nil
}

func (c *mediaControl) Events() <-chan session.Event {
	return c.events
}

func (c *mediaControl) Send(ctx context.Context, command session.Command) error {
	if ctx == nil {
		return fmt.Errorf("media control context is required")
	}
	if err := command.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("marshal media command: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := c.connection.Write(ctx, websocket.MessageText, payload); err != nil {
		return fmt.Errorf("send media command: %w", err)
	}
	return nil
}

func (c *mediaControl) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.cancel()
		err = c.connection.Close(websocket.StatusNormalClosure, "control closed")
	})
	return err
}

func (c *mediaControl) readLoop(ctx context.Context) {
	defer close(c.events)
	for {
		kind, payload, err := c.connection.Read(ctx)
		if err != nil {
			return
		}
		if kind != websocket.MessageText {
			return
		}
		var event session.Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return
		}
		select {
		case c.events <- event:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runtime) registerControl(
	ctx context.Context,
	call sqlc.Call,
	sessionID uuid.UUID,
	rawURL string,
) error {
	r.mu.Lock()
	state := r.states[sessionID]
	r.mu.Unlock()
	if state == nil {
		record, _, err := r.orchestrator.AttachCall(ctx, call.OrganizationID, call.ID, *call.VoiceAgentID)
		if err != nil {
			return err
		}
		turns, err := r.orchestrator.ListTurns(ctx, call.OrganizationID, sessionID)
		if err != nil {
			return err
		}
		state = restoreConversationState(record, turns)
	}
	control, err := r.media.OpenControl(ctx, rawURL)
	if err != nil {
		return err
	}

	r.mu.Lock()
	if existing := r.controls[sessionID]; existing != nil {
		r.mu.Unlock()
		_ = control.Close()
		return fmt.Errorf("media control already registered for session %s", sessionID)
	}
	r.controls[sessionID] = control
	if r.states[sessionID] == nil {
		r.states[sessionID] = state
	}
	r.callSessions[call.ID] = sessionID
	r.mu.Unlock()

	go r.observeControl(call, sessionID, control)
	return nil
}

func (r *Runtime) observeControl(call sqlc.Call, sessionID uuid.UUID, control *mediaControl) {
	defer func() {
		r.mu.Lock()
		if current := r.controls[sessionID]; current == control {
			delete(r.controls, sessionID)
		}
		r.mu.Unlock()
		_ = control.Close()
	}()

	for event := range control.Events() {
		r.mu.Lock()
		state := r.states[sessionID]
		r.mu.Unlock()
		if state != nil {
			state.observe(event)
		}
		if r.logger != nil {
			r.logger.Info(
				context.Background(),
				"Voice Agent media event",
				"call_id", call.ID,
				"voice_agent_session_id", sessionID,
				"event_type", event.Type,
				"provider_id", event.ProviderID,
			)
		}
		switch event.Type {
		case session.EventTranscriptFinal:
			if state != nil {
				if turn, ok := state.userTurn(event); ok {
					r.persistTurn(call.OrganizationID, sessionID, turn)
				}
			}
		case session.EventResponseStopped:
			if state != nil {
				if turn, ok := state.assistantTurn(event); ok {
					r.persistTurn(call.OrganizationID, sessionID, turn)
				}
			}
		case session.EventToolCall:
			if event.ToolCall != nil {
				go r.executeRealtimeTool(call, sessionID, control, *event.ToolCall)
				continue
			}
		}
		if event.Type == session.EventError &&
			event.Failure != nil &&
			event.Failure.Terminal {
			endedAt := event.OccurredAt
			if endedAt.IsZero() {
				endedAt = time.Now().UTC()
			}
			_ = r.failSession(context.Background(), call, endedAt)
			return
		}
	}
}

func (r *Runtime) stopMediaSession(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	control := r.controls[id]
	if control != nil {
		delete(r.controls, id)
	}
	r.mu.Unlock()

	node, placed := r.mediaNode(ctx, id)
	if control != nil {
		err := control.Send(ctx, session.Command{Type: session.CommandStop})
		closeErr := control.Close()
		if err == nil {
			r.releaseMediaNode(context.Background(), id)
			return closeErr
		}
	}

	var err error
	if placed {
		err = r.media.StopSessionAt(ctx, node.ControlURL, id)
	} else {
		err = r.media.StopSession(ctx, id)
	}
	r.releaseMediaNode(context.Background(), id)
	return err
}

func (r *Runtime) executeRealtimeTool(
	call sqlc.Call,
	sessionID uuid.UUID,
	control *mediaControl,
	toolCall session.ToolCallEvent,
) {
	if call.VoiceAgentID == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	result := session.ToolResult{
		ToolCallID: toolCall.ID,
		Name:       toolCall.Name,
	}
	execution, err := r.orchestrator.ExecuteTool(ctx, tools.ExecuteRequest{
		OrganizationID: call.OrganizationID,
		VoiceAgentID:   *call.VoiceAgentID,
		SessionID:      sessionID,
		CallID:         call.ID,
		ToolName:       toolCall.Name,
		ToolCallID:     toolCall.ID,
		Arguments:      toolCall.Arguments,
	})
	if err == nil {
		result.Name = execution.Name
		result.Content = string(execution.Body)
	}
	if err != nil {
		result.IsError = true
		result.Content = err.Error()
	}
	if result.Content == "" {
		result.Content = "{}"
	}

	r.mu.Lock()
	state := r.states[sessionID]
	r.mu.Unlock()
	if state != nil {
		r.persistTurn(
			call.OrganizationID,
			sessionID,
			state.toolTurn(result.Name, result.ToolCallID, result.Content, result.IsError),
		)
	}

	if sendErr := control.Send(ctx, session.Command{
		Type:       session.CommandToolResult,
		ToolResult: &result,
	}); sendErr != nil && r.logger != nil {
		r.logger.Error(
			context.Background(),
			"send Voice Agent tool result",
			"call_id", call.ID,
			"voice_agent_session_id", sessionID,
			"tool_call_id", toolCall.ID,
			"tool_name", toolCall.Name,
			"error", sendErr,
		)
	}
}

func (r *Runtime) persistTurn(
	organizationID, sessionID uuid.UUID,
	req conversations.CreateTurnRequest,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := r.orchestrator.CreateTurn(ctx, organizationID, sessionID, req); err != nil && r.logger != nil {
		r.logger.Error(
			context.Background(),
			"persist Voice Agent turn",
			"voice_agent_session_id", sessionID,
			"sequence", req.Sequence,
			"role", req.Role,
			"error", err,
		)
	}
}

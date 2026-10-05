package agent

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/coffeyvidzro/monogo/internal/runtime/medianodes"
)

func (r *Runtime) resumeControl(ctx context.Context, call sqlc.Call, record sqlc.VoiceAgentSession) error {
	r.mu.Lock()
	connected := r.controls[record.ID] != nil
	r.mu.Unlock()
	if connected {
		return nil
	}
	base := r.media.baseURL
	if r.mediaNodes != nil {
		node, err := r.mediaNodes.Owner(ctx, record.ID)
		if errors.Is(err, medianodes.ErrOwnerUnavailable) {
			return &controlHTTPError{status: 404, err: err}
		}
		if err != nil {
			return err
		}
		base = node.ControlURL
	}
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return err
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path += "/internal/v1/sessions/" + record.ID.String() + "/control"
	return r.registerControl(ctx, call, record.ID, u.String())
}

// RunRecovery reconnects live calls after worker loss or a control socket failure.
// It never creates a replacement audio fork for an already attached session.
func (r *Runtime) RunRecovery(ctx context.Context, queries *sqlc.Queries, channels *calling.ChannelStore) error {
	defer r.closeControls()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := r.recoverCalls(ctx, queries, channels); err != nil && r.logger != nil {
			r.logger.Error(ctx, "recover Voice Agent calls", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runtime) recoverCalls(ctx context.Context, queries *sqlc.Queries, channels *calling.ChannelStore) error {
	calls, err := queries.ListCallsForVoiceAgentRecovery(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, call := range calls {
		callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		channelID, err := channels.Get(callCtx, call.ID)
		if err == nil {
			err = r.attach(callCtx, call, channelID)
			var controlErr *controlHTTPError
			if errors.As(err, &controlErr) && controlErr.status == 404 {
				// Media is gone; do not leave an answered but silent phone call.
				err = errors.Join(err, r.failSession(callCtx, call, time.Now().UTC()), r.freeSwitch.Hangup(callCtx, channelID))
			}
		}
		cancel()
		if err != nil {
			failures = append(failures, fmt.Errorf("call %s: %w", call.ID, err))
		}
	}
	return errors.Join(failures...)
}

func (r *Runtime) closeControls() {
	r.mu.Lock()
	controls := make([]*mediaControl, 0, len(r.controls))
	for _, control := range r.controls {
		controls = append(controls, control)
	}
	r.mu.Unlock()
	for _, control := range controls {
		_ = control.Close()
	}
}

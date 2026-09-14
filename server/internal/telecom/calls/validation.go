package calls

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type controlAction string

const (
	controlAnswer   controlAction = "answer"
	controlHangup   controlAction = "hang up"
	controlTransfer controlAction = "transfer"
	controlHold     controlAction = "hold"
	controlUnhold   controlAction = "resume"
	controlPlay     controlAction = "play audio"
	controlStop     controlAction = "stop audio"
	controlRecord   controlAction = "record"
	controlDTMF     controlAction = "send DTMF"
)

func validateOrganizationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	return nil
}

func require(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperror.NewBadRequest(field + " is required")
	}
	return value, nil
}

func validateCreateRequest(req *CreateCallRequest) error {
	if req.ApplicationID != nil && *req.ApplicationID == uuid.Nil {
		return apperror.NewBadRequest("application_id is invalid")
	}
	if req.TrunkID != nil && *req.TrunkID == uuid.Nil {
		return apperror.NewBadRequest("trunk_id is invalid")
	}

	var err error
	if req.From, err = require(req.From, "from"); err != nil {
		return err
	}
	if req.To, err = require(req.To, "to"); err != nil {
		return err
	}
	if req.DTMFMode == "" {
		req.DTMFMode = "rfc2833"
	}
	req.DTMFMode = strings.ToLower(strings.TrimSpace(req.DTMFMode))
	if req.DTMFMode != "rfc2833" && req.DTMFMode != "info" {
		return apperror.NewBadRequest("dtmf_mode must be rfc2833 or info")
	}
	if len(req.Codecs) == 0 {
		req.Codecs = []string{"PCMU", "PCMA", "G722", "OPUS"}
	}
	if len(req.Codecs) > 8 {
		return apperror.NewBadRequest("codecs cannot contain more than 8 entries")
	}
	allowedCodecs := map[string]struct{}{"PCMU": {}, "PCMA": {}, "G722": {}, "OPUS": {}}
	seen := make(map[string]struct{}, len(req.Codecs))
	for index, codec := range req.Codecs {
		codec = strings.ToUpper(strings.TrimSpace(codec))
		if _, ok := allowedCodecs[codec]; !ok {
			return apperror.NewBadRequest("unsupported codec: " + codec)
		}
		if _, ok := seen[codec]; ok {
			return apperror.NewBadRequest("duplicate codec: " + codec)
		}
		seen[codec] = struct{}{}
		req.Codecs[index] = codec
	}
	if req.MediaEncryption == "" {
		req.MediaEncryption = "rtp"
	}
	req.MediaEncryption = strings.ToLower(strings.TrimSpace(req.MediaEncryption))
	if req.MediaEncryption != "rtp" && req.MediaEncryption != "sdes_srtp" {
		return apperror.NewBadRequest("media_encryption must be rtp or sdes_srtp")
	}
	return nil
}

func validateRecordRequest(req *RecordRequest) error {
	if req.Action == "" {
		req.Action = "start"
	}
	if req.Action != "start" && req.Action != "stop" {
		return apperror.NewBadRequest("action must be start or stop")
	}
	if req.Action == "start" {
		var err error
		req.Path, err = require(req.Path, "path")
		return err
	}
	return nil
}

func validateControl(call sqlc.Call, action controlAction) error {
	if action == controlAnswer {
		if call.Direction != string(DirectionInbound) {
			return apperror.NewConflict("outbound call cannot be answered")
		}
		if canAnswer(call.State) || isAnswerIdempotent(call.State) {
			return nil
		}
		return invalidControlState(action, call.State)
	}

	if isConnected(call.State) {
		return nil
	}
	return invalidControlState(action, call.State)
}

func validateMediaState(call sqlc.Call) error {
	switch CallMediaState(call.MediaState) {
	case MediaStateActive, MediaStateHeld:
		return nil
	default:
		return apperror.NewConflict("call has invalid media state: " + call.MediaState)
	}
}

func invalidControlState(action controlAction, state string) error {
	return apperror.NewConflict(fmt.Sprintf("cannot %s call from state: %s", action, state))
}

func canAnswer(state string) bool {
	return state == string(StateInitiating) || state == string(StateRinging)
}

func isAnswerIdempotent(state string) bool {
	return state == string(StateAnswered) || state == string(StateActive)
}

func isConnected(state string) bool {
	return state == string(StateAnswered) || state == string(StateActive)
}

func isPreAnswer(state string) bool {
	return state == string(StateInitiating) || state == string(StateRinging)
}

func isTerminal(state string) bool {
	return state == string(StateCompleted) || state == string(StateFailed) || state == string(StateCancelled)
}

func isHeld(call sqlc.Call) bool {
	return CallMediaState(call.MediaState) == MediaStateHeld
}

func isMediaActive(call sqlc.Call) bool {
	return CallMediaState(call.MediaState) == MediaStateActive
}

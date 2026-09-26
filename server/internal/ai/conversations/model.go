// Package conversations owns durable AI call sessions and turn history.
package conversations

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CompleteRequest struct {
	State                  string
	TurnCount              int32
	InterruptionCount      int32
	FirstResponseLatencyMS *int32
	AverageTurnLatencyMS   *int32
	EndedAt                time.Time
}

type CreateTurnRequest struct {
	Sequence        int32
	Role            string
	Content         string
	ProviderID      *string
	ToolName        *string
	ToolCallID      *string
	Metadata        json.RawMessage
	SpeechStartedAt *time.Time
	SpeechEndedAt   *time.Time
	STTLatencyMS    *int32
	LLMTTFTMS       *int32
	TTSTTFBMS       *int32
	TurnLatencyMS   *int32
}

type Identity struct {
	OrganizationID uuid.UUID
	SessionID      uuid.UUID
}

package tools

import (
	"encoding/json"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func toolFromSnapshot(snapshot []byte, id uuid.UUID, name string) (sqlc.VoiceAgentTool, error) {
	var definitions []struct {
		ID uuid.UUID `json:"id"`
		Name string `json:"name"`
		Description string `json:"description"`
		Parameters json.RawMessage `json:"parameters"`
		Type string `json:"type"`
		EndpointURL *string `json:"endpoint_url"`
		TimeoutMS int32 `json:"timeout_ms"`
	}
	if err := json.Unmarshal(snapshot, &definitions); err != nil {
		return sqlc.VoiceAgentTool{}, apperror.NewForbidden("invalid session tool snapshot")
	}
	name = strings.TrimSpace(name)
	for _, tool := range definitions {
		if (id == uuid.Nil || tool.ID == id) && (name == "" || tool.Name == name) {
			if tool.ID == uuid.Nil || tool.Type == "" || tool.TimeoutMS <= 0 {
				return sqlc.VoiceAgentTool{}, apperror.NewForbidden("session tool snapshot is incomplete; reactivate the agent for new calls")
			}
			return sqlc.VoiceAgentTool{
				ID: tool.ID,
				Name: tool.Name,
				Description: tool.Description,
				Parameters: []byte(tool.Parameters),
				Type: tool.Type,
				EndpointUrl: tool.EndpointURL,
				TimeoutMs: tool.TimeoutMS,
			}, nil
		}
	}
	return sqlc.VoiceAgentTool{}, apperror.NewForbidden("tool is not available in this session")
}

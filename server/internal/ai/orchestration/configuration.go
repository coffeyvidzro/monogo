package orchestration

import (
	"encoding/json"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func MediaConfig(agent sqlc.VoiceAgent, config session.Config) session.Config {
	config.Engine = session.Engine(agent.Engine)
	config.Instructions = agent.Instructions
	config.EngineConfig = json.RawMessage(append([]byte(nil), agent.EngineConfig...))
	if agent.Voice != nil {
		config.Voice = *agent.Voice
	}
	if agent.Language != nil {
		config.Language = *agent.Language
	}
	return config
}

// MediaConfigFromSession builds live media configuration only from the durable
// session snapshot. Editing an agent after answer must not change an active call.
func MediaConfigFromSession(record sqlc.VoiceAgentSession, config session.Config) session.Config {
	config.Engine = session.Engine(record.Engine)
	config.Instructions = record.InstructionsSnapshot
	config.EngineConfig = json.RawMessage(append([]byte(nil), record.EngineConfigSnapshot...))
	if record.Voice != nil {
		config.Voice = *record.Voice
	}
	if record.Language != nil {
		config.Language = *record.Language
	}
	return config
}

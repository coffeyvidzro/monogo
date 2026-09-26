package orchestration

import (
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func MediaConfig(agent sqlc.VoiceAgent, config session.Config) session.Config {
	config.Engine = session.Engine(agent.Engine)
	config.Instructions = agent.Instructions
	if agent.Voice != nil {
		config.Voice = *agent.Voice
	}
	if agent.Language != nil {
		config.Language = *agent.Language
	}
	return config
}

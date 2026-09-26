package voiceai

import (
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/ai/orchestration"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
)

type Runtime struct {
	orchestrator *orchestration.Service
	media        *mediaClient
	freeSwitch   *freeswitch.Client
}

func New(
	orchestrator *orchestration.Service,
	freeSwitch *freeswitch.Client,
	cfg Config,
) (*Runtime, error) {
	if orchestrator == nil {
		return nil, fmt.Errorf("voice AI orchestrator is required")
	}
	if freeSwitch == nil {
		return nil, fmt.Errorf("voice AI FreeSWITCH client is required")
	}
	media, err := newMediaClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		orchestrator: orchestrator,
		media:        media,
		freeSwitch:   freeSwitch,
	}, nil
}

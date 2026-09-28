package voiceai

import (
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/ai/orchestration"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Runtime struct {
	orchestrator *orchestration.Service
	media        *mediaClient
	freeSwitch   *freeswitch.Client
	logger       *logging.Logger
	db           *pgxpool.Pool
	nats         *natsintegration.Client
}

func New(
	orchestrator *orchestration.Service,
	freeSwitch *freeswitch.Client,
	cfg Config,
	loggers ...*logging.Logger,
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
	var logger *logging.Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	return &Runtime{
		orchestrator: orchestrator,
		media:        media,
		freeSwitch:   freeSwitch,
		logger:       logger,
	}, nil
}

package agent

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/coffeyvidzro/monogo/internal/ai/orchestration"
	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/runtime/medianodes"
)

type Runtime struct {
	orchestrator *orchestration.Service
	media        *mediaClient
	freeSwitch   *freeswitch.Client
	logger       *logging.Logger
	mediaNodes   *medianodes.Registry

	attachMu     sync.Mutex
	mu           sync.Mutex
	controls     map[uuid.UUID]*mediaControl
	states       map[uuid.UUID]*conversationState
	callSessions map[uuid.UUID]uuid.UUID
	sessionNodes map[uuid.UUID]medianodes.Node
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
	return newRuntime(orchestrator, freeSwitch, media, logger, nil), nil
}

func NewWithMediaNodes(
	orchestrator *orchestration.Service,
	freeSwitch *freeswitch.Client,
	cfg Config,
	registry *medianodes.Registry,
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
	return newRuntime(orchestrator, freeSwitch, media, logger, registry), nil
}

func newRuntime(
	orchestrator *orchestration.Service,
	freeSwitch *freeswitch.Client,
	media *mediaClient,
	logger *logging.Logger,
	registry *medianodes.Registry,
) *Runtime {
	return &Runtime{
		orchestrator: orchestrator,
		media:        media,
		freeSwitch:   freeSwitch,
		logger:       logger,
		mediaNodes:   registry,
		controls:     make(map[uuid.UUID]*mediaControl),
		states:       make(map[uuid.UUID]*conversationState),
		callSessions: make(map[uuid.UUID]uuid.UUID),
		sessionNodes: make(map[uuid.UUID]medianodes.Node),
	}
}

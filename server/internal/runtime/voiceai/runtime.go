package voiceai

import "github.com/coffeyvidzro/monogo/internal/ai/orchestration"

type Runtime struct{ orchestrator *orchestration.Service }

func New(orchestrator *orchestration.Service) *Runtime { return &Runtime{orchestrator: orchestrator} }

package orchestration

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/ai/tools"
)

func (s *Service) ExecuteTool(
	ctx context.Context,
	req tools.ExecuteRequest,
) (tools.ExecuteResult, error) {
	return s.tools.Execute(ctx, req)
}

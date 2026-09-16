package worker

import (
	"context"
	"fmt"
)

func (m *modules) ready(ctx context.Context) error {
	if err := m.postgres.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if err := m.nats.Ping(ctx); err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	return nil
}

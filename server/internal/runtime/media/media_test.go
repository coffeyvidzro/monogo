package media

import (
	"context"
	"testing"
)

func TestRunStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunRejectsNilContext(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatal("Run() error = nil")
	}
}

package calls

import (
	"context"
	"fmt"
)

type Consumer struct {
	service *Service
}

func NewConsumer(service *Service) *Consumer {
	if service == nil {
		panic("calls: service is required")
	}
	return &Consumer{service: service}
}

func (c *Consumer) HandleInbound(
	ctx context.Context,
	req InboundAdmissionRequest,
) error {
	if _, err := c.service.AdmitInbound(ctx, req); err != nil {
		return fmt.Errorf("admit inbound call: %w", err)
	}
	return nil
}

func (c *Consumer) HandleLifecycle(
	ctx context.Context,
	event LifecycleEvent,
) error {
	if err := c.service.ObserveLifecycle(ctx, event); err != nil {
		return fmt.Errorf("handle call lifecycle event: %w", err)
	}
	return nil
}

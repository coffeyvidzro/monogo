package webhooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

func validOrg(v uuid.UUID) error {
	if v == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	return nil
}

func validIDs(org, id uuid.UUID) error {
	if err := validOrg(org); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("webhook id is required")
	}
	return nil
}

func validateInboundEvent(event InboundEvent) error {
	if event.ID == uuid.Nil {
		return fmt.Errorf("webhook event id is required")
	}
	if event.OrganizationID == uuid.Nil {
		return fmt.Errorf("webhook event organization id is required")
	}
	if strings.TrimSpace(event.EventType) == "" {
		return fmt.Errorf("webhook event type is required")
	}
	if strings.TrimSpace(event.ObjectType) == "" {
		return fmt.Errorf("webhook event object type is required")
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("webhook event occurred_at is required")
	}
	if !json.Valid(event.Payload) || len(bytes.TrimSpace(event.Payload)) == 0 || bytes.TrimSpace(event.Payload)[0] != '{' {
		return fmt.Errorf("webhook event payload must be a JSON object")
	}
	return nil
}

func normalizeURL(v string) (string, error) {
	v = strings.TrimSpace(v)
	u, err := url.ParseRequestURI(v)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", apperror.NewBadRequest("url must be an https URL")
	}
	return v, nil
}

func normalizeEvents(v []string) ([]string, error) {
	if len(v) == 0 {
		return nil, apperror.NewBadRequest("subscribed_events is required")
	}
	out := make([]string, 0, len(v))
	seen := map[string]struct{}{}
	for _, event := range v {
		event = strings.TrimSpace(event)
		if event == "" || strings.ContainsAny(event, " \t\r\n") {
			return nil, apperror.NewBadRequest("subscribed_events must contain non-empty event names")
		}
		if _, ok := seen[event]; ok {
			continue
		}
		seen[event] = struct{}{}
		out = append(out, event)
	}
	return out, nil
}

func normalizeCreate(r *CreateRequest) error {
	var err error
	r.URL, err = normalizeURL(r.URL)
	if err != nil {
		return err
	}
	r.SubscribedEvents, err = normalizeEvents(r.SubscribedEvents)
	return err
}

func normalizeUpdate(r *UpdateRequest) error {
	if r.URL != nil {
		value, err := normalizeURL(*r.URL)
		if err != nil {
			return err
		}
		r.URL = &value
	}
	if r.SubscribedEvents != nil {
		value, err := normalizeEvents(*r.SubscribedEvents)
		if err != nil {
			return err
		}
		r.SubscribedEvents = &value
	}
	return nil
}

package runtimes

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestResponseReportsFreshRuntimeOnline(t *testing.T) {
	now := time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC)
	lastSeen := now.Add(-10 * time.Second)
	value := Runtime{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Name:           "edge-a",
		Status:         StatusActive,
		Capabilities:   []string{},
		LastSeenAt:     &lastSeen,
	}
	if !response(value, now).Online {
		t.Fatal("fresh active runtime reported offline")
	}
}

func TestResponseReportsStaleRuntimeOffline(t *testing.T) {
	now := time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC)
	lastSeen := now.Add(-31 * time.Second)
	value := Runtime{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Name:           "edge-a",
		Status:         StatusActive,
		Capabilities:   []string{},
		LastSeenAt:     &lastSeen,
	}
	if response(value, now).Online {
		t.Fatal("stale runtime reported online")
	}
}

func TestResponseReportsDisabledRuntimeOffline(t *testing.T) {
	now := time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC)
	lastSeen := now
	value := Runtime{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Name:           "edge-a",
		Status:         StatusDisabled,
		Capabilities:   []string{},
		LastSeenAt:     &lastSeen,
	}
	if response(value, now).Online {
		t.Fatal("disabled runtime reported online")
	}
}

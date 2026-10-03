package runtimes

import "testing"

func TestNormalizeCreate(t *testing.T) {
	region := " gh-accra "
	req := CreateRequest{Name: " Edge Runtime ", Region: &region}
	if err := normalizeCreate(&req); err != nil {
		t.Fatalf("normalizeCreate() error = %v", err)
	}
	if req.Name != "Edge Runtime" {
		t.Fatalf("name = %q", req.Name)
	}
	if req.Region == nil || *req.Region != "gh-accra" {
		t.Fatalf("region = %#v", req.Region)
	}
}

func TestNormalizeHeartbeat(t *testing.T) {
	req := HeartbeatRequest{
		Version:        " 1.2.3 ",
		Capabilities:   []string{"Realtime", "voice", "realtime"},
		Capacity:       100,
		ActiveSessions: 12,
	}
	if err := normalizeHeartbeat(&req); err != nil {
		t.Fatalf("normalizeHeartbeat() error = %v", err)
	}
	if req.Version != "1.2.3" {
		t.Fatalf("version = %q", req.Version)
	}
	if len(req.Capabilities) != 2 || req.Capabilities[0] != "realtime" || req.Capabilities[1] != "voice" {
		t.Fatalf("capabilities = %#v", req.Capabilities)
	}
}

func TestNormalizeHeartbeatRejectsSessionCountAboveCapacity(t *testing.T) {
	req := HeartbeatRequest{
		Version:        "1.2.3",
		Capacity:       10,
		ActiveSessions: 11,
	}
	if err := normalizeHeartbeat(&req); err == nil {
		t.Fatal("normalizeHeartbeat() error = nil, want capacity error")
	}
}

func TestNormalizeHeartbeatRejectsInvalidCapability(t *testing.T) {
	req := HeartbeatRequest{
		Version:      "1.2.3",
		Capabilities: []string{"voice agent"},
		Capacity:     10,
	}
	if err := normalizeHeartbeat(&req); err == nil {
		t.Fatal("normalizeHeartbeat() error = nil, want capability error")
	}
}

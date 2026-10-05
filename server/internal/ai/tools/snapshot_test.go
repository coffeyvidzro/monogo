package tools

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestToolSnapshotPinsIdentityAndExecutableConfiguration(t *testing.T) {
	id := uuid.New()
	value := []byte(`[{"id":"` + id.String() + `","name":"lookup","description":"Lookup","parameters":{"type":"object"},"type":"webhook","endpoint_url":"https://original.example/tool","timeout_ms":3000}]`)
	tool, err := toolFromSnapshot(value, uuid.Nil, "lookup")
	if err != nil {
		t.Fatal(err)
	}
	if tool.ID != id || tool.EndpointUrl == nil || *tool.EndpointUrl != "https://original.example/tool" || tool.TimeoutMs != 3000 || !json.Valid(tool.Parameters) {
		t.Fatalf("unexpected snapshot tool: %+v", tool)
	}
	for _, request := range []struct {
		id   uuid.UUID
		name string
	}{
		{uuid.New(), "lookup"}, {id, "replacement"}, {uuid.Nil, "new_tool"},
	} {
		if _, err := toolFromSnapshot(value, request.id, request.name); err == nil {
			t.Fatal("tool outside session capability set accepted")
		}
	}
}

func TestLegacyToolSnapshotFailsClosed(t *testing.T) {
	value := []byte(`[{"id":"` + uuid.NewString() + `","name":"lookup","parameters":{}}]`)
	if _, err := toolFromSnapshot(value, uuid.Nil, "lookup"); err == nil {
		t.Fatal("legacy snapshot accepted mutable execution configuration")
	}
}

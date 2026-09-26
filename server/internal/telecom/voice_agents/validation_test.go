package voice_agents

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeCreate(t *testing.T) {
	req, err := normalizeCreate(CreateRequest{
		Name:         "  Support Agent  ",
		Engine:       EngineComposable,
		Instructions: "  Help the caller.  ",
	})
	if err != nil {
		t.Fatalf("normalizeCreate() error = %v", err)
	}
	if req.Name != "Support Agent" || req.Instructions != "Help the caller." {
		t.Fatalf("normalized request = %+v", req)
	}
}

func TestNormalizeCreateRejectsInvalidEngine(t *testing.T) {
	_, err := normalizeCreate(CreateRequest{
		Name: "Support Agent", Engine: "unknown", Instructions: "Help the caller.",
	})
	if err == nil {
		t.Fatal("normalizeCreate() error = nil")
	}
}

func TestNormalizeUpdateRequiresField(t *testing.T) {
	if _, err := normalizeUpdate(UpdateRequest{}); err == nil {
		t.Fatal("normalizeUpdate() error = nil")
	}
}

func TestValidateBindingRequiresApplication(t *testing.T) {
	if err := validateBinding(CreateBindingRequest{}); err == nil {
		t.Fatal("validateBinding() error = nil")
	}
	if err := validateBinding(CreateBindingRequest{VoiceApplicationID: uuid.New()}); err != nil {
		t.Fatalf("validateBinding() error = %v", err)
	}
}

package agents

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeCreate(t *testing.T) {
	req, err := normalizeCreate(CreateRequest{
		Name: "  Support Agent  ",
		Engine: EngineComposable,
		Instructions: "  Help the caller.  ",
	})
	if err != nil { t.Fatalf("normalizeCreate() error = %v", err) }
	if req.Name != "Support Agent" || req.Instructions != "Help the caller." {
		t.Fatalf("normalized request = %+v", req)
	}
}

func TestNormalizeCreateRejectsInvalidEngine(t *testing.T) {
	_, err := normalizeCreate(CreateRequest{Name:"Support Agent",Engine:"unknown",Instructions:"Help the caller."})
	if err == nil { t.Fatal("normalizeCreate() error = nil") }
}

func TestCreateBindingRequiresApplication(t *testing.T) {
	if uuid.Nil != (uuid.UUID{}) {
		t.Fatal("unexpected uuid zero value")
	}
}

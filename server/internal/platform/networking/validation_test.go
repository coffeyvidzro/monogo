package networking

import "testing"

func TestValidateCreateCanonicalizesCIDR(t *testing.T) {
	req, prefix, err := validateCreate(CreateRequest{
		Name:       " Office ",
		Action:     ActionAllow,
		SourceCIDR: "192.0.2.42/24",
	})
	if err != nil {
		t.Fatalf("validateCreate() error = %v", err)
	}
	if req.Name != "Office" || prefix.String() != "192.0.2.0/24" {
		t.Fatalf("validateCreate() = (%q, %q)", req.Name, prefix)
	}
}

func TestValidateCreateRejectsInvalidPolicy(t *testing.T) {
	_, _, err := validateCreate(CreateRequest{
		Name:       "Office",
		Action:     "permit",
		SourceCIDR: "not-a-cidr",
	})
	if err == nil {
		t.Fatal("validateCreate() error = nil")
	}
}

func TestValidateUpdateNormalizesEveryField(t *testing.T) {
	name := " Office "
	action := "allow"
	status := "disabled"
	source := "192.0.2.42/24"
	req, prefix, err := validateUpdate(UpdateRequest{
		Name:       &name,
		Action:     &action,
		SourceCIDR: &source,
		Status:     &status,
	})
	if err != nil {
		t.Fatalf("validateUpdate() error = %v", err)
	}
	if *req.Name != "Office" || *req.SourceCIDR != "192.0.2.0/24" || prefix.String() != "192.0.2.0/24" {
		t.Fatalf("validateUpdate() = (%q, %q, %q)", *req.Name, *req.SourceCIDR, prefix)
	}
}

func TestValidateUpdateRejectsEmptyAndInvalidFields(t *testing.T) {
	if _, _, err := validateUpdate(UpdateRequest{}); err == nil {
		t.Fatal("validateUpdate(empty) error = nil")
	}
	status := "pending"
	if _, _, err := validateUpdate(UpdateRequest{
		Status: &status,
	}); err == nil {
		t.Fatal("validateUpdate(invalid status) error = nil")
	}
}

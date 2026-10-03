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

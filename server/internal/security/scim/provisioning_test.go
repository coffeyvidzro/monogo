package scim

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeUserInput(t *testing.T) {
	input, active, err := normalizeUserInput(UserInput{
		UserName:    "  PERSON@EXAMPLE.COM ",
		DisplayName: " Person Example ",
	})
	if err != nil {
		t.Fatalf("normalizeUserInput() error = %v", err)
	}
	if input.UserName != "person@example.com" || input.DisplayName != "Person Example" || !active {
		t.Fatalf("normalized user = %#v, active = %v", input, active)
	}
}

func TestNormalizeUserInputRejectsNonEmailUserName(t *testing.T) {
	if _, _, err := normalizeUserInput(UserInput{UserName: "not-an-email"}); err == nil {
		t.Fatal("normalizeUserInput() expected error")
	}
}

func TestNormalizeGroupInputDeduplicatesMembers(t *testing.T) {
	id := uuid.New()
	input, ids, err := normalizeGroupInput(GroupInput{
		DisplayName: "Support",
		Members: []GroupMember{
			{Value: id.String()},
			{Value: id.String()},
		},
	})
	if err != nil {
		t.Fatalf("normalizeGroupInput() error = %v", err)
	}
	if input.DisplayName != "Support" || len(ids) != 1 || ids[0] != id {
		t.Fatalf("normalized group = %#v, ids = %v", input, ids)
	}
}

func TestEqualityFilter(t *testing.T) {
	value, ok, err := equalityFilter(`userName eq "person@example.com"`, "userName")
	if err != nil {
		t.Fatalf("equalityFilter() error = %v", err)
	}
	if !ok || value != "person@example.com" {
		t.Fatalf("filter = %q, %v", value, ok)
	}
}

func TestEqualityFilterRejectsUnsupportedAttribute(t *testing.T) {
	if _, _, err := equalityFilter(`emails.value eq "person@example.com"`, "userName"); err == nil {
		t.Fatal("equalityFilter() expected unsupported filter error")
	}
}

func TestPageClampsCount(t *testing.T) {
	start, limit, offset := page(2, maxPageCount+10)
	if start != 2 || limit != maxPageCount || offset != 1 {
		t.Fatalf("page() = %d, %d, %d", start, limit, offset)
	}
}

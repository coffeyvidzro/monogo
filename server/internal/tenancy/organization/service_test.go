package organization

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type stubServiceRepository struct {
	member  sqlc.OrganizationMember
	updated bool
	deleted bool
}

func (*stubServiceRepository) CreateWithOwner(context.Context, string, uuid.UUID) (sqlc.Organization, error) {
	return sqlc.Organization{}, nil
}
func (*stubServiceRepository) GetByID(context.Context, uuid.UUID) (sqlc.Organization, error) {
	return sqlc.Organization{}, nil
}
func (s *stubServiceRepository) GetMember(context.Context, sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	return s.member, nil
}
func (s *stubServiceRepository) Update(_ context.Context, arg sqlc.UpdateOrganizationParams) (sqlc.Organization, error) {
	s.updated = true
	return sqlc.Organization{ID: arg.ID, Name: "updated", Status: "active"}, nil
}
func (s *stubServiceRepository) Delete(context.Context, uuid.UUID) error {
	s.deleted = true
	return nil
}
func (*stubServiceRepository) ListByUserID(context.Context, uuid.UUID) ([]sqlc.ListOrganizationsByUserIDRow, error) {
	return nil, nil
}

func TestUpdateRequiresOwnerOrAdmin(t *testing.T) {
	for _, tt := range []struct {
		role    string
		allowed bool
	}{
		{role: ownerRole, allowed: true},
		{role: adminRole, allowed: true},
		{role: "member", allowed: false},
	} {
		t.Run(tt.role, func(t *testing.T) {
			repo := &stubServiceRepository{member: sqlc.OrganizationMember{Role: tt.role}}
			name := "updated"
			_, err := NewService(repo).Update(context.Background(), uuid.New(), uuid.New(), UpdateRequest{Name: &name})
			assertRoleResult(t, err, repo.updated, tt.allowed)
		})
	}
}

func TestDeleteRequiresOwner(t *testing.T) {
	for _, tt := range []struct {
		role    string
		allowed bool
	}{{role: ownerRole, allowed: true}, {role: adminRole, allowed: false}, {role: "member", allowed: false}} {
		t.Run(tt.role, func(t *testing.T) {
			repo := &stubServiceRepository{member: sqlc.OrganizationMember{Role: tt.role}}
			err := NewService(repo).Delete(context.Background(), uuid.New(), uuid.New())
			assertRoleResult(t, err, repo.deleted, tt.allowed)
		})
	}
}

func assertRoleResult(t *testing.T, err error, called, allowed bool) {
	t.Helper()
	if allowed {
		if err != nil || !called {
			t.Fatalf("expected operation to succeed, called=%t err=%v", called, err)
		}
		return
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusForbidden || called {
		t.Fatalf("expected forbidden without repository mutation, called=%t err=%v", called, err)
	}
}

package members

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type stubRepository struct {
	actorID    uuid.UUID
	addArg     sqlc.AddOrganizationMemberParams
	updateArg  sqlc.UpdateMemberRoleParams
	disableArg sqlc.DisableOrganizationMemberParams
}

func (s *stubRepository) Add(_ context.Context, arg sqlc.AddOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	s.addArg = arg
	return sqlc.OrganizationMember{OrganizationID: arg.OrganizationID, UserID: arg.UserID, Role: arg.Role}, nil
}
func (s *stubRepository) Get(_ context.Context, arg sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	role := roleMember
	if arg.UserID == s.actorID {
		role = roleAdmin
	}
	return sqlc.OrganizationMember{OrganizationID: arg.OrganizationID, UserID: arg.UserID, Role: role}, nil
}
func (*stubRepository) ListByOrganizationID(context.Context, uuid.UUID) ([]sqlc.OrganizationMember, error) {
	return nil, nil
}
func (s *stubRepository) UpdateRole(_ context.Context, arg sqlc.UpdateMemberRoleParams) (sqlc.OrganizationMember, error) {
	s.updateArg = arg
	return sqlc.OrganizationMember{OrganizationID: arg.OrganizationID, UserID: arg.UserID, Role: arg.Role}, nil
}
func (s *stubRepository) Disable(_ context.Context, arg sqlc.DisableOrganizationMemberParams) error {
	s.disableArg = arg
	return nil
}

func TestMemberMutationsPropagateActorUserID(t *testing.T) {
	actorID, organizationID, targetID := uuid.New(), uuid.New(), uuid.New()
	repo := &stubRepository{actorID: actorID}
	service := NewService(repo)

	if _, err := service.Add(context.Background(), actorID, organizationID, CreateRequest{UserID: targetID, Role: roleMember}); err != nil {
		t.Fatalf("Add returned an error: %v", err)
	}
	if repo.addArg.ActorUserID != actorID {
		t.Fatalf("Add actor = %s, want %s", repo.addArg.ActorUserID, actorID)
	}

	if _, err := service.Update(context.Background(), actorID, organizationID, targetID, UpdateRequest{Role: roleAdmin}); err != nil {
		t.Fatalf("Update returned an error: %v", err)
	}
	if repo.updateArg.ActorUserID != actorID {
		t.Fatalf("Update actor = %s, want %s", repo.updateArg.ActorUserID, actorID)
	}

	if err := service.Delete(context.Background(), actorID, organizationID, targetID); err != nil {
		t.Fatalf("Delete returned an error: %v", err)
	}
	if repo.disableArg.ActorUserID != actorID {
		t.Fatalf("Delete actor = %s, want %s", repo.disableArg.ActorUserID, actorID)
	}
}

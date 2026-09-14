package members

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type recordingDB struct {
	args []any
}

func (d *recordingDB) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	d.args = args
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (*recordingDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}
func (d *recordingDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	d.args = args
	return emptyRow{}
}

type emptyRow struct{}

func (emptyRow) Scan(...any) error { return nil }

func TestRepositoryPassesActorToMutationQueries(t *testing.T) {
	organizationID, targetID, actorID := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name  string
		run   func(*Repository) error
		index int
	}{
		{name: "add", index: 3, run: func(repo *Repository) error {
			_, err := repo.Add(context.Background(), sqlc.AddOrganizationMemberParams{
				OrganizationID: organizationID, UserID: targetID, Role: roleMember, ActorUserID: actorID,
			})
			return err
		}},
		{name: "update", index: 3, run: func(repo *Repository) error {
			_, err := repo.UpdateRole(context.Background(), sqlc.UpdateMemberRoleParams{
				OrganizationID: organizationID, UserID: targetID, Role: roleAdmin, ActorUserID: actorID,
			})
			return err
		}},
		{name: "disable", index: 2, run: func(repo *Repository) error {
			return repo.Disable(context.Background(), sqlc.DisableOrganizationMemberParams{
				OrganizationID: organizationID, UserID: targetID, ActorUserID: actorID,
			})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &recordingDB{}
			repo := NewRepository(sqlc.New(db))
			if err := tt.run(repo); err != nil {
				t.Fatalf("repository mutation returned an error: %v", err)
			}
			if got := db.args[tt.index]; got != actorID {
				t.Fatalf("actor query argument = %v, want %s", got, actorID)
			}
		})
	}
}

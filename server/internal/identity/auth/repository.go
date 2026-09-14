package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) GetUserByEmail(
	ctx context.Context,
	email string,
) (sqlc.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

func (r *Repository) GetUserByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *Repository) CreateUser(
	ctx context.Context,
	arg sqlc.CreateUserParams,
) (sqlc.User, error) {
	return r.queries.CreateUser(ctx, arg)
}

func (r *Repository) SetUserPassword(
	ctx context.Context,
	arg sqlc.SetUserPasswordParams,
) (sqlc.User, error) {
	return r.queries.SetUserPassword(ctx, arg)
}

func (r *Repository) MarkUserEmailVerified(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.User, error) {
	return r.queries.MarkUserEmailVerified(ctx, id)
}

func (r *Repository) CreateAuthTransaction(
	ctx context.Context,
	arg sqlc.CreateAuthTransactionParams,
) (sqlc.AuthTransaction, error) {
	return r.queries.CreateAuthTransaction(ctx, arg)
}

func (r *Repository) GetAuthTransactionByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.AuthTransaction, error) {
	return r.queries.GetAuthTransactionByID(ctx, id)
}

func (r *Repository) SetAuthTransactionMethod(
	ctx context.Context,
	arg sqlc.SetAuthTransactionMethodParams,
) (sqlc.AuthTransaction, error) {
	return r.queries.SetAuthTransactionMethod(ctx, arg)
}

func (r *Repository) SetAuthTransactionState(
	ctx context.Context,
	arg sqlc.SetAuthTransactionStateParams,
) (sqlc.AuthTransaction, error) {
	return r.queries.SetAuthTransactionState(ctx, arg)
}

func (r *Repository) SetAuthTransactionUser(
	ctx context.Context,
	arg sqlc.SetAuthTransactionUserParams,
) (sqlc.AuthTransaction, error) {
	return r.queries.SetAuthTransactionUser(ctx, arg)
}

func (r *Repository) MarkAuthTransactionAuthenticated(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.AuthTransaction, error) {
	return r.queries.MarkAuthTransactionAuthenticated(ctx, id)
}

func (r *Repository) ExpireAuthTransaction(
	ctx context.Context,
	id uuid.UUID,
) error {
	return r.queries.ExpireAuthTransaction(ctx, id)
}

func (r *Repository) CreateAuthChallenge(
	ctx context.Context,
	arg sqlc.CreateAuthChallengeParams,
) (sqlc.AuthChallenge, error) {
	return r.queries.CreateAuthChallenge(ctx, arg)
}

func (r *Repository) GetActiveAuthChallenge(
	ctx context.Context,
	arg sqlc.GetActiveAuthChallengeParams,
) (sqlc.AuthChallenge, error) {
	return r.queries.GetActiveAuthChallenge(ctx, arg)
}

func (r *Repository) IncrementAuthChallengeAttempts(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.AuthChallenge, error) {
	return r.queries.IncrementAuthChallengeAttempts(ctx, id)
}

func (r *Repository) ConsumeAuthChallenge(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.AuthChallenge, error) {
	return r.queries.ConsumeAuthChallenge(ctx, id)
}

package auth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/security/otp"
	"github.com/leamout/leamout/internal/security/password"
	"github.com/leamout/leamout/internal/security/token"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Start(ctx context.Context, email string) (sqlc.AuthTransaction, error) {
	email = normalizeEmail(email)
	if email == "" {
		return sqlc.AuthTransaction{}, apperror.NewBadRequest("email is required")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return sqlc.AuthTransaction{}, err
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	transaction, err := s.repo.CreateAuthTransaction(ctx, sqlc.CreateAuthTransactionParams{
		UserID:     &user.ID,
		Identifier: email,
		ExpiresAt:  pgconv.NullableTimestamptz(&expiresAt),
	})
	if err != nil {
		return sqlc.AuthTransaction{}, err
	}

	return transaction, nil
}

// LoginWithPassword authenticates a user using the password associated with
// the authentication transaction.
func (s *Service) LoginWithPassword(ctx context.Context, transactionID uuid.UUID, value string) (sqlc.User, error) {
	transaction, err := s.getValidTransaction(ctx, transactionID)
	if err != nil {
		return sqlc.User{}, err
	}

	if transaction.UserID == nil {
		return sqlc.User{}, apperror.NewUnauthorized("invalid credentials")
	}

	user, err := s.repo.GetUserByID(ctx, *transaction.UserID)
	if err != nil {
		return sqlc.User{}, apperror.NewUnauthorized("invalid credentials")
	}

	if user.DisabledAt.Valid {
		return sqlc.User{}, apperror.NewUnauthorized("account is disabled")
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		return sqlc.User{}, apperror.NewBadRequest("password is not enrolled")
	}

	if !password.Verify(value, *user.PasswordHash) {
		return sqlc.User{}, apperror.NewUnauthorized("invalid credentials")
	}

	if _, err := s.repo.MarkAuthTransactionAuthenticated(ctx, transactionID); err != nil {
		return sqlc.User{}, err
	}

	return user, nil
}

func (s *Service) SetPassword(ctx context.Context, userID uuid.UUID, value string) (sqlc.User, error) {
	if userID == uuid.Nil {
		return sqlc.User{}, apperror.NewUnauthorized("authentication required")
	}

	hash, err := password.Hash(value)
	if err != nil {
		return sqlc.User{}, apperror.NewInternal("failed to hash password", err)
	}

	return s.repo.SetUserPassword(ctx, sqlc.SetUserPasswordParams{
		ID:           userID,
		PasswordHash: &hash,
	})
}

func (s *Service) SendOTP(ctx context.Context, transactionID uuid.UUID) (string, error) {
	transaction, err := s.getValidTransaction(ctx, transactionID)
	if err != nil {
		return "", err
	}

	code, err := otp.GenerateNumeric(6)
	if err != nil {
		return "", apperror.NewInternal("failed to generate authentication code", err)
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	_, err = s.repo.CreateAuthChallenge(ctx, sqlc.CreateAuthChallengeParams{
		Identifier:        transaction.Identifier,
		SecretHash:        token.Hash(code),
		ExpiresAt:         pgconv.NullableTimestamptz(&expiresAt),
		Purpose:           "otp",
		AuthTransactionID: &transactionID,
		MaxAttempts:       5,
	})
	if err != nil {
		return "", err
	}

	return code, nil
}

// VerifyOTP verifies the active OTP challenge and authenticates the
// corresponding transaction.
func (s *Service) VerifyOTP(ctx context.Context, transactionID uuid.UUID, code string) (sqlc.User, error) {
	transaction, err := s.getValidTransaction(ctx, transactionID)
	if err != nil {
		return sqlc.User{}, err
	}

	challenge, err := s.repo.GetActiveAuthChallenge(ctx, sqlc.GetActiveAuthChallengeParams{
		AuthTransactionID: &transactionID,
		Purpose:           "otp",
	})
	if err != nil {
		return sqlc.User{}, apperror.NewUnauthorized("invalid or expired authentication code")
	}

	if challenge.ConsumedAt.Valid {
		return sqlc.User{}, apperror.NewUnauthorized("authentication code has already been used")
	}

	if pgconv.TimestamptzToTime(challenge.ExpiresAt).Before(time.Now()) {
		return sqlc.User{}, apperror.NewUnauthorized("authentication code has expired")
	}

	if challenge.Attempts >= 5 {
		return sqlc.User{}, apperror.NewUnauthorized("too many authentication attempts")
	}

	if !token.Verify(code, challenge.SecretHash) {
		_, _ = s.repo.IncrementAuthChallengeAttempts(ctx, challenge.ID)
		return sqlc.User{}, apperror.NewUnauthorized("invalid authentication code")
	}

	if _, err := s.repo.ConsumeAuthChallenge(ctx, challenge.ID); err != nil {
		return sqlc.User{}, err
	}

	user, err := s.getOrCreateOTPUser(ctx, transaction)
	if err != nil {
		return sqlc.User{}, err
	}

	if user.DisabledAt.Valid {
		return sqlc.User{}, apperror.NewUnauthorized("account is disabled")
	}

	if _, err := s.repo.MarkUserEmailVerified(ctx, user.ID); err != nil {
		return sqlc.User{}, err
	}

	if _, err := s.repo.MarkAuthTransactionAuthenticated(ctx, transactionID); err != nil {
		return sqlc.User{}, err
	}

	return user, nil
}

func (s *Service) getValidTransaction(ctx context.Context, transactionID uuid.UUID) (sqlc.AuthTransaction, error) {
	if transactionID == uuid.Nil {
		return sqlc.AuthTransaction{}, apperror.NewBadRequest("invalid transaction_id")
	}

	transaction, err := s.repo.GetAuthTransactionByID(ctx, transactionID)
	if err != nil {
		return sqlc.AuthTransaction{}, apperror.NewUnauthorized("invalid authentication transaction")
	}

	if transaction.ExpiresAt.Valid && pgconv.TimestamptzToTime(transaction.ExpiresAt).Before(time.Now()) {
		_ = s.repo.ExpireAuthTransaction(ctx, transactionID)
		return sqlc.AuthTransaction{}, apperror.NewUnauthorized("authentication transaction has expired")
	}

	return transaction, nil
}

func (s *Service) getOrCreateOTPUser(ctx context.Context, transaction sqlc.AuthTransaction) (sqlc.User, error) {
	if transaction.UserID != nil {
		return s.repo.GetUserByID(ctx, *transaction.UserID)
	}

	email := normalizeEmail(transaction.Identifier)
	if strings.TrimSpace(email) == "" {
		return sqlc.User{}, apperror.NewBadRequest("authentication identifier is missing")
	}

	return s.repo.CreateUser(ctx, sqlc.CreateUserParams{Email: email})
}

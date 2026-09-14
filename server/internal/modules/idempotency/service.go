package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Service struct {
	repository *Repository
	config     Config
	now        func() time.Time
}

func NewService(repository *Repository, config Config) *Service {
	return &Service{repository: repository, config: config, now: time.Now}
}

func (s *Service) Claim(ctx context.Context, request Request) (Claim, error) {
	now := s.now().UTC()
	record, claimed, err := s.repository.Claim(ctx, sqlc.ClaimIdempotencyKeyParams{
		Scope: request.Scope, IdempotencyKey: request.Key, Method: request.Method,
		Path: request.Path, RequestHash: request.RequestHash,
		LockedUntil: pgconv.NullableTimestamptz(timePointer(now.Add(s.config.LockTTL))),
		ExpiresAt:   pgconv.NullableTimestamptz(timePointer(now.Add(s.config.RecordTTL))),
	})
	if err != nil {
		return Claim{}, err
	}
	if claimed {
		return Claim{Lease: pgconv.TimestamptzToTime(record.LockedUntil)}, nil
	}
	if record.Method != request.Method || record.Path != request.Path || record.RequestHash != request.RequestHash {
		return Claim{}, ErrKeyConflict
	}
	if record.Status != "completed" {
		return Claim{}, ErrInProgress
	}

	headers := map[string][]string{}
	if err := json.Unmarshal(record.ResponseHeaders, &headers); err != nil {
		return Claim{}, err
	}
	return Claim{Response: &Response{
		Status: int(*record.ResponseStatus), Body: record.ResponseBody,
		ContentType: stringValue(record.ResponseContentType), Headers: headers,
	}}, nil
}

func (s *Service) Complete(ctx context.Context, request Request, lease time.Time, response Response) error {
	headers, err := json.Marshal(response.Headers)
	if err != nil {
		return err
	}
	contentType := response.ContentType
	status := int32(response.Status)
	err = s.repository.Complete(ctx, sqlc.CompleteIdempotencyKeyParams{
		ResponseStatus: &status, ResponseBody: response.Body,
		ResponseContentType: &contentType, ResponseHeaders: headers,
		Scope: request.Scope, IdempotencyKey: request.Key, Method: request.Method,
		Path: request.Path, RequestHash: request.RequestHash,
		LockedUntil: pgconv.NullableTimestamptz(timePointer(lease)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInProgress
	}
	return err
}

func timePointer(value time.Time) *time.Time { return &value }

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) WithTx(tx pgx.Tx) *Service {
	return &Service{
		repo: s.repo.WithTx(tx),
		now:  s.now,
	}
}

func (s *Service) WithTx(tx pgx.Tx) *Service {
	return &Service{
		repo: s.repo.WithTx(tx),
		now:  s.now,
	}
}

func (s *Service) CreateAttempt(
	ctx context.Context,
	req CreateAttemptRequest,
) (Payment, error) {
	if err := validateCreateAttemptRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	existing, err := s.repo.GetActiveByCheckout(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
	)
	if err == nil {
		if !sameAttempt(existing, req) {
			return Payment{}, apperror.NewConflict("active payment attempt conflicts with checkout")
		}
		return paymentFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewInternal("get active payment attempt", err)
	}

	nextAttempt, err := s.repo.NextAttempt(
		ctx,
		req.OrganizationID,
		req.CheckoutID,
	)
	if err != nil {
		return Payment{}, apperror.NewInternal("get next payment attempt", err)
	}
	if nextAttempt <= 0 || nextAttempt > math.MaxInt32 {
		return Payment{}, apperror.NewInternal("create payment attempt", ErrConflict)
	}

	row, err := s.repo.CreateAttempt(
		ctx,
		req,
		int32(nextAttempt),
	)
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("payment attempt conflict")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("create payment attempt", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) AttachProviderPaymentID(
	ctx context.Context,
	req AttachProviderPaymentIDRequest,
) (Payment, error) {
	if err := validateAttachProviderPaymentIDRequest(&req); err != nil {
		return Payment{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.AttachProviderPaymentID(ctx, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment attempt does not allow provider payment id")
	}
	if isUniqueViolation(err) {
		return Payment{}, apperror.NewConflict("provider payment id already belongs to another payment")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("attach provider payment id", err)
	}

	return paymentFromRow(row), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func paymentFromRow(row sqlc.Payment) Payment {
	return Payment{
		ID:                row.ID,
		CheckoutID:        row.CheckoutID,
		OrganizationID:    row.OrganizationID,
		Provider:          row.Provider,
		PaymentMethod:     row.PaymentMethod,
		Attempt:           row.Attempt,
		AmountMicros:      row.AmountMicros,
		Currency:          row.Currency,
		Status:            row.Status,
		ProviderPaymentID: row.ProviderPaymentID,
		FailureCode:       row.FailureCode,
		PaidAt:            pgconv.TimestamptzToTimePtr(row.PaidAt),
		CreatedAt:         pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:         pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

func (s *Service) RecordProviderEvent(
	ctx context.Context,
	req RecordProviderEventRequest,
) (ProviderEvent, Payment, error) {
	if err := validateRecordProviderEventRequest(&req); err != nil {
		return ProviderEvent{}, Payment{}, apperror.NewBadRequest(err.Error())
	}
	if req.ReceivedAt.IsZero() {
		req.ReceivedAt = s.now().UTC()
	}

	paymentRow, err := s.repo.GetByProviderPaymentID(
		ctx,
		req.Provider,
		req.ProviderPaymentID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderEvent{}, Payment{}, apperror.NewNotFound("payment not found")
	}
	if err != nil {
		return ProviderEvent{}, Payment{}, apperror.NewInternal("get payment by provider identity", err)
	}

	sum := sha256.Sum256(req.Payload)
	payloadSHA256 := hex.EncodeToString(sum[:])
	eventRow, err := s.repo.CreateProviderEvent(
		ctx,
		paymentRow,
		req,
		payloadSHA256,
		req.ReceivedAt,
	)
	if isUniqueViolation(err) {
		existing, getErr := s.repo.GetProviderEventByIdentity(
			ctx,
			req.Provider,
			req.ProviderEventID,
		)
		if getErr != nil {
			return ProviderEvent{}, Payment{}, apperror.NewInternal("get provider event replay", getErr)
		}
		if existing.PaymentID != paymentRow.ID ||
			existing.PayloadSha256 != payloadSHA256 {
			return ProviderEvent{}, Payment{}, apperror.NewConflict("provider event payload does not match original event")
		}

		return providerEventFromRow(existing), paymentFromRow(paymentRow), nil
	}
	if err != nil {
		return ProviderEvent{}, Payment{}, apperror.NewInternal("record provider event", err)
	}

	return providerEventFromRow(eventRow), paymentFromRow(paymentRow), nil
}

func (s *Service) MarkProviderEventProcessed(
	ctx context.Context,
	eventID uuid.UUID,
	processedAt time.Time,
) (ProviderEvent, error) {
	if eventID == uuid.Nil {
		return ProviderEvent{}, apperror.NewBadRequest("provider event id is required")
	}
	if processedAt.IsZero() {
		processedAt = s.now().UTC()
	}

	row, err := s.repo.MarkProviderEventProcessed(ctx, eventID, processedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderEvent{}, apperror.NewConflict("provider event is already processed")
	}
	if err != nil {
		return ProviderEvent{}, apperror.NewInternal("mark provider event processed", err)
	}

	return providerEventFromRow(row), nil
}

func providerEventFromRow(row sqlc.PaymentProviderEvent) ProviderEvent {
	return ProviderEvent{
		ID:              row.ID,
		PaymentID:       row.PaymentID,
		OrganizationID:  row.OrganizationID,
		Provider:        row.Provider,
		ProviderEventID: row.ProviderEventID,
		EventType:       row.EventType,
		PayloadSHA256:   row.PayloadSha256,
		Payload:         row.Payload,
		ReceivedAt:      pgconv.TimestamptzToTime(row.ReceivedAt),
		ProcessedAt:     pgconv.TimestamptzToTimePtr(row.ProcessedAt),
	}
}

func (s *Service) MarkSucceeded(
	ctx context.Context,
	payment Payment,
	paidAt time.Time,
) (Payment, error) {
	if payment.Status == StatusSucceeded {
		return payment, nil
	}
	if payment.Status != StatusPending && payment.Status != StatusProcessing {
		return Payment{}, apperror.NewConflict("payment attempt cannot succeed")
	}
	if paidAt.IsZero() {
		paidAt = s.now().UTC()
	}

	row, err := s.repo.MarkSucceeded(
		ctx,
		payment.OrganizationID,
		payment.CheckoutID,
		payment.ID,
		paidAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment attempt cannot succeed")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("mark payment attempt succeeded", err)
	}

	return paymentFromRow(row), nil
}

func (s *Service) MarkFailed(
	ctx context.Context,
	payment Payment,
	failureCode string,
) (Payment, error) {
	if payment.Status == StatusFailed && payment.FailureCode != nil &&
		*payment.FailureCode == failureCode {
		return payment, nil
	}
	if payment.Status != StatusPending && payment.Status != StatusProcessing {
		return Payment{}, apperror.NewConflict("payment attempt cannot fail")
	}
	if failureCode == "" {
		return Payment{}, apperror.NewBadRequest("failure code is required")
	}

	row, err := s.repo.MarkFailed(
		ctx,
		payment.OrganizationID,
		payment.CheckoutID,
		payment.ID,
		failureCode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, apperror.NewConflict("payment attempt cannot fail")
	}
	if err != nil {
		return Payment{}, apperror.NewInternal("mark payment attempt failed", err)
	}

	return paymentFromRow(row), nil
}


func sameAttempt(
	existing sqlc.Payment,
	req CreateAttemptRequest,
) bool {
	return existing.CheckoutID == req.CheckoutID &&
		existing.OrganizationID == req.OrganizationID &&
		existing.Provider == req.Provider &&
		existing.PaymentMethod == req.PaymentMethod &&
		existing.AmountMicros == req.AmountMicros &&
		existing.Currency == req.Currency
}

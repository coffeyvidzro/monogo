package numbers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redisv9 "github.com/redis/go-redis/v9"

	"github.com/leamout/leamout/internal/database/sqlc"
	redisintegration "github.com/leamout/leamout/internal/integrations/redis"
	"github.com/leamout/leamout/internal/modules/audit"
)

const managedNumberSelectionTTL = 10 * time.Minute

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
	redis   *redisintegration.Client
}

func NewRepository(db *pgxpool.Pool, redis *redisintegration.Client) *Repository {
	return &Repository{db: db, queries: sqlc.New(db), redis: redis}
}

func (r *Repository) SaveManagedSelection(ctx context.Context, organizationID uuid.UUID, candidate ManagedNumberCandidate) (string, error) {
	if r == nil || r.redis == nil {
		return "", fmt.Errorf("managed number selection store is unavailable")
	}
	if organizationID == uuid.Nil {
		return "", fmt.Errorf("organization id is required")
	}
	selectionID := "sel_" + uuid.NewString()
	if err := r.redis.SetJSON(ctx, selectionKey(organizationID, selectionID), candidate, managedNumberSelectionTTL); err != nil {
		return "", fmt.Errorf("store managed number selection: %w", err)
	}
	return selectionID, nil
}

func (r *Repository) LoadManagedSelection(ctx context.Context, organizationID uuid.UUID, selectionID string) (ManagedNumberCandidate, error) {
	return r.loadSelection(ctx, organizationID, selectionID)
}

func (r *Repository) CreateBYOC(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.PhoneNumber, error) {
	return r.queries.CreateBYOCPhoneNumber(ctx, sqlc.CreateBYOCPhoneNumberParams{
		OrganizationID:      organizationID,
		Number:              req.Number,
		CountryCode:         req.CountryCode,
		CarrierConnectionID: req.CarrierConnectionID,
		VoiceEnabled:        req.VoiceEnabled,
		SmsEnabled:          req.SMSEnabled,
	})
}

func (r *Repository) CreateManaged(
	ctx context.Context,
	organizationID uuid.UUID,
	selectionID string,
	authorization ManagedNumberPurchaseAuthorization,
) (sqlc.PhoneNumber, error) {
	selection, err := r.loadSelection(ctx, organizationID, selectionID)
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)

	provider, err := queries.GetCarrierProviderBySlug(ctx, selection.Provider)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.PhoneNumber{}, ErrSelectionUnavailable
	}
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}

	carrierConnectionID, providerRoutingResourceID, err := resolveProviderRoutingTarget(ctx, queries, provider.ID)
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}

	number, err := queries.CreateProvisioningManagedPhoneNumber(ctx, sqlc.CreateProvisioningManagedPhoneNumberParams{
		OrganizationID:      organizationID,
		Number:              selection.Number,
		CountryCode:         selection.CountryCode,
		ProviderID:          provider.ID,
		CarrierConnectionID: carrierConnectionID,
	})
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}

	request, err := json.Marshal(ProviderOperationRequest{
		Provider:                  selection.Provider,
		ProviderInventoryID:       selection.ProviderInventoryID,
		ProviderProductID:         selection.ProviderProductID,
		Number:                    selection.Number,
		CountryCode:               selection.CountryCode,
		CarrierConnectionID:       carrierConnectionID,
		ProviderRoutingResourceID: providerRoutingResourceID,
		PurchaseAuthorization:     authorization,
	})
	if err != nil {
		return sqlc.PhoneNumber{}, fmt.Errorf("encode provider operation request: %w", err)
	}
	if _, err := queries.CreateNumberProvisionProviderOperation(ctx, sqlc.CreateNumberProvisionProviderOperationParams{
		OrganizationID:    organizationID,
		CarrierProviderID: provider.ID,
		PhoneNumberID:     number.ID,
		IdempotencyKey:    "number-provision:" + number.ID.String(),
		Request:           request,
	}); err != nil {
		return sqlc.PhoneNumber{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	_ = r.redis.Delete(ctx, selectionKey(organizationID, selectionID))
	return number, nil
}

func resolveProviderRoutingTarget(ctx context.Context, queries *sqlc.Queries, providerID uuid.UUID) (uuid.UUID, string, error) {
	targets, err := queries.ListProviderRoutingTargets(ctx, providerID)
	if err != nil {
		return uuid.Nil, "", err
	}
	if len(targets) != 1 {
		return uuid.Nil, "", ErrProviderRoutingUnavailable
	}
	return targets[0].CarrierConnectionID, targets[0].ProviderResourceID, nil
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.PhoneNumber, error) {
	return r.queries.ListPhoneNumbersByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberByID(ctx, sqlc.GetPhoneNumberByIDParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) GetForRelease(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberForRelease(ctx, sqlc.GetPhoneNumberForReleaseParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.PhoneNumber, error) {
	return r.queries.UpdatePhoneNumber(ctx, sqlc.UpdatePhoneNumberParams{
		ID: id, OrganizationID: organizationID, VoiceEnabled: req.VoiceEnabled, SmsEnabled: req.SMSEnabled,
	})
}

func (r *Repository) ReleaseBYOC(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.ReleaseBYOCPhoneNumber(ctx, sqlc.ReleaseBYOCPhoneNumberParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) SetCarrierConnection(ctx context.Context, organizationID, id, connectionID uuid.UUID, event audit.Event) (sqlc.PhoneNumber, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	number, err := r.queries.WithTx(tx).SetBYOCPhoneNumberCarrierConnection(ctx, sqlc.SetBYOCPhoneNumberCarrierConnectionParams{
		ID: id, OrganizationID: organizationID, CarrierConnectionID: &connectionID,
	})
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if err := audit.Insert(ctx, tx, event); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	return number, nil
}

func (r *Repository) ListProviderOperationsReady(ctx context.Context, limit int32) ([]sqlc.ProviderOperation, error) {
	return r.queries.ListProviderOperationsReadyForRetry(ctx, limit)
}

func (r *Repository) TryProviderOperationLock(ctx context.Context, id uuid.UUID) (func(), bool, error) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	queries := sqlc.New(conn)
	locked, err := queries.TryProviderOperationAdvisoryLock(ctx, id.String())
	if err != nil {
		conn.Release()
		return nil, false, err
	}
	if !locked {
		conn.Release()
		return nil, false, nil
	}
	return func() {
		_ = queries.ReleaseProviderOperationAdvisoryLock(context.Background(), id.String())
		conn.Release()
	}, true, nil
}

func (r *Repository) MarkProviderOperationAccepted(ctx context.Context, operation sqlc.ProviderOperation, providerOperationID string, response []byte) error {
	_, err := r.queries.MarkNumberProvisionProviderOperationAccepted(ctx, sqlc.MarkNumberProvisionProviderOperationAcceptedParams{
		ProviderOperationID: &providerOperationID,
		Response:            response,
		ID:                  operation.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) && operation.State == "provider_accepted" {
		return nil
	}
	return err
}

func (r *Repository) RecordProviderOperationFailure(ctx context.Context, id uuid.UUID, cause error) error {
	if cause == nil {
		return nil
	}
	message := cause.Error()
	_, err := r.queries.RecordProviderOperationAttemptFailure(ctx, sqlc.RecordProviderOperationAttemptFailureParams{
		LastError: &message,
		ID:        id,
	})
	return err
}

func (r *Repository) FailProviderOperation(ctx context.Context, operation sqlc.ProviderOperation, cause error) error {
	message := "managed number provisioning failed"
	if cause != nil {
		message = cause.Error()
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	if err := queries.MarkProviderOperationFailed(ctx, sqlc.MarkProviderOperationFailedParams{LastError: &message, ID: operation.ID}); err != nil {
		return err
	}
	if err := queries.MarkManagedPhoneNumberFailed(ctx, sqlc.MarkManagedPhoneNumberFailedParams{
		ErrorMessage:   &message,
		ID:             operation.PhoneNumberID,
		OrganizationID: operation.OrganizationID,
		ProviderID:     &operation.CarrierProviderID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) CompleteProviderOperation(ctx context.Context, operation sqlc.ProviderOperation, request ProviderOperationRequest, providerResourceID string, response []byte) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)

	number, err := queries.LockManagedPhoneNumberForProviderOperation(ctx, sqlc.LockManagedPhoneNumberForProviderOperationParams{
		ID:             operation.PhoneNumberID,
		OrganizationID: operation.OrganizationID,
		ProviderID:     &operation.CarrierProviderID,
		Number:         request.Number,
		CountryCode:    request.CountryCode,
	})
	if err != nil {
		return err
	}
	if number.Status != "provisioning" {
		return fmt.Errorf("managed phone number is not provisioning")
	}

	resourceID := providerResourceID
	if _, err := queries.MarkManagedPhoneNumberActive(ctx, sqlc.MarkManagedPhoneNumberActiveParams{
		ProviderResourceID: &resourceID,
		ID:                 number.ID,
		OrganizationID:     number.OrganizationID,
		ProviderID:         &operation.CarrierProviderID,
	}); err != nil {
		return err
	}
	if _, err := queries.MarkNumberProvisionProviderOperationSucceeded(ctx, sqlc.MarkNumberProvisionProviderOperationSucceededParams{
		ProviderResourceID: &resourceID,
		Response:           response,
		ID:                 operation.ID,
		OrganizationID:     operation.OrganizationID,
		CarrierProviderID:  operation.CarrierProviderID,
		PhoneNumberID:      number.ID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) loadSelection(ctx context.Context, organizationID uuid.UUID, selectionID string) (ManagedNumberCandidate, error) {
	if r == nil || r.redis == nil {
		return ManagedNumberCandidate{}, fmt.Errorf("managed number selection store is unavailable")
	}
	var selection ManagedNumberCandidate
	if err := r.redis.GetJSON(ctx, selectionKey(organizationID, selectionID), &selection); err != nil {
		if errors.Is(err, redisv9.Nil) {
			return ManagedNumberCandidate{}, ErrSelectionNotFound
		}
		return ManagedNumberCandidate{}, fmt.Errorf("load managed number selection: %w", err)
	}
	return selection, nil
}

func selectionKey(organizationID uuid.UUID, selectionID string) string {
	return "telecom:numbers:selection:" + organizationID.String() + ":" + selectionID
}

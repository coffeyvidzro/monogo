package numbers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo           *Repository
	inventory      *didww.Client
	provider       managedProvider
	inboundTrunkID string
	db             *pgxpool.Pool
	now            func() time.Time
}

func NewService(repository *Repository, inventory *didww.Client) *Service {
	return &Service{repo: repository, inventory: inventory, provider: inventory, now: time.Now}
}

func (s *Service) ConfigureManaged(db *pgxpool.Pool, inboundTrunkID string) {
	s.db = db
	s.inboundTrunkID = strings.TrimSpace(inboundTrunkID)
}

// SearchAvailable reads DIDWW inventory only. It cannot reserve, purchase, or
// provision a number and never exposes upstream resource identifiers.
func (s *Service) SearchAvailable(
	ctx context.Context,
	organizationID uuid.UUID,
	contains string,
) ([]AvailableNumber, error) {
	if err := validateOrganization(organizationID); err != nil {
		return nil, err
	}
	contains = strings.TrimSpace(contains)
	if len(contains) < 3 || len(contains) > 15 {
		return nil, apperror.NewBadRequest("contains must contain 3 to 15 digits")
	}
	for _, digit := range contains {
		if digit < '0' || digit > '9' {
			return nil, apperror.NewBadRequest("contains must contain only digits")
		}
	}
	if s.inventory == nil {
		return nil, apperror.NewServiceUnavailable("managed number inventory is not configured", nil)
	}

	result, err := s.inventory.SearchAvailableDIDs(ctx, didww.AvailableDIDFilter{
		NumberContains: contains,
	})
	if err != nil {
		return nil, apperror.NewServiceUnavailable("managed number inventory is unavailable", err)
	}

	numbers := make([]AvailableNumber, 0, len(result.Data))
	seen := make(map[string]struct{}, len(result.Data))
	for _, availableDID := range result.Data {
		number := strings.TrimSpace(availableDID.Attributes.Number)
		if !strings.HasPrefix(number, "+") {
			number = "+" + number
		}
		if !e164.MatchString(number) || !strings.Contains(number, contains) {
			continue
		}
		if _, exists := seen[number]; exists {
			continue
		}

		seen[number] = struct{}{}
		numbers = append(numbers, AvailableNumber{Number: number})
	}
	return numbers, nil
}

func (s *Service) CreateBYOC(ctx context.Context, organizationID uuid.UUID, req CreateBYOCRequest) (sqlc.PhoneNumber, error) {
	if err := validateOrganization(organizationID); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if err := normalizeBYOC(&req); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	row, err := s.repo.CreateBYOC(ctx, organizationID, req)
	return row, writeError(err)
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.PhoneNumber, error) {
	if err := validateOrganization(organizationID); err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list numbers", err)
	}
	return rows, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	row, err := s.repo.Get(ctx, organizationID, id)
	return row, readError(err)
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.PhoneNumber, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if err := validateUpdate(req); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	row, err := s.repo.Update(ctx, organizationID, id, req)
	return row, writeError(err)
}

func (s *Service) SetBYOCConnection(ctx context.Context, organizationID, id uuid.UUID, req SetCarrierConnectionRequest) (sqlc.PhoneNumber, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if req.CarrierConnectionID == uuid.Nil {
		return sqlc.PhoneNumber{}, apperror.NewBadRequest("carrier_connection_id is required")
	}
	row, err := s.repo.SetBYOCConnection(ctx, organizationID, id, req.CarrierConnectionID)
	return row, writeError(err)
}

func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	number, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if number.ProvisioningMode != "byoc" {
		return apperror.NewConflict("managed number release requires provider deprovisioning")
	}
	_, err = s.repo.ReleaseBYOC(ctx, organizationID, id)
	return writeError(err)
}

func readError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("number not found")
	}
	return apperror.NewInternal("get number", err)
}

func writeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("number or carrier connection not found")
	}
	var dbError *pgconn.PgError
	if errors.As(err, &dbError) {
		switch dbError.Code {
		case "23505":
			return apperror.NewConflict("number already exists")
		case "23503", "23514", "23502":
			return apperror.NewBadRequest("number or carrier connection is invalid")
		}
	}
	return apperror.NewInternal("update number", err)
}

const managedReconcileDelay = 30 * time.Second

type managedProvider interface {
	SearchAvailableDIDs(context.Context, didww.AvailableDIDFilter) (didww.AvailableDIDList, error)
	OrderDID(context.Context, didww.OrderDIDRequest) (didww.Order, error)
	GetOrder(context.Context, string) (didww.Order, error)
	FindOrderByExternalReference(context.Context, string) (didww.Order, bool, error)
	FindDIDByNumber(context.Context, string) (didww.DID, error)
	AssignDIDVoiceInTrunk(context.Context, string, string) (didww.DID, error)
	GetDID(context.Context, string) (didww.DID, error)
}

func (s *Service) Purchase(ctx context.Context, organizationID uuid.UUID, key string, req ManagedPurchaseRequest) (ManagedOrder, error) {
	key = strings.TrimSpace(key)
	if err := normalizeManagedPurchase(organizationID, key, &req); err != nil {
		return ManagedOrder{}, err
	}
	digest := sha256.Sum256([]byte(req.Number + "\n" + req.CountryCode))
	requestHash := hex.EncodeToString(digest[:])
	existing, err := s.repo.GetManagedOrderByKey(ctx, organizationID, key)
	if err == nil {
		if existing.RequestHash != requestHash {
			return ManagedOrder{}, apperror.NewConflict("idempotency key was used with a different purchase")
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ManagedOrder{}, apperror.NewInternal("read managed number order", err)
	}
	if s.provider == nil || s.inboundTrunkID == "" {
		return ManagedOrder{}, apperror.NewServiceUnavailable("managed number purchasing is not configured", nil)
	}

	// Resolve the provider identity again at purchase time. Search results are
	// display-only and are never treated as proof that inventory is still valid.
	inventoryResult, err := s.provider.SearchAvailableDIDs(ctx, didww.AvailableDIDFilter{NumberContains: strings.TrimPrefix(req.Number, "+")})
	if err != nil {
		return ManagedOrder{}, apperror.NewConflict("number is no longer available")
	}
	var inventory didww.AvailableDID
	for _, candidate := range inventoryResult.Data {
		if normalizeProviderNumber(candidate.Attributes.Number) == req.Number {
			if inventory.ID != "" {
				return ManagedOrder{}, apperror.NewServiceUnavailable("provider returned ambiguous inventory", nil)
			}
			inventory = candidate
		}
	}
	if inventory.ID == "" {
		return ManagedOrder{}, apperror.NewConflict("number is no longer available")
	}
	providerNumber := normalizeProviderNumber(inventory.Attributes.Number)
	if providerNumber != req.Number {
		return ManagedOrder{}, apperror.NewConflict("provider inventory does not match requested number")
	}
	skuID, err := availableDIDSKU(inventory)
	if err != nil {
		return ManagedOrder{}, apperror.NewServiceUnavailable("provider inventory is incomplete", err)
	}

	order, created, err := s.repo.CreateManagedOrder(
		ctx,
		organizationID,
		key,
		requestHash,
		req,
		inventory.ID,
		skuID,
	)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	if !created {
		if order.RequestHash != requestHash {
			return ManagedOrder{}, apperror.NewConflict("idempotency key was used with a different purchase")
		}
		return order, nil
	}
	return s.submit(ctx, order)
}

func (s *Service) submit(ctx context.Context, order ManagedOrder) (ManagedOrder, error) {
	claimed, err := s.repo.ClaimManagedSubmission(ctx, order.ID)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	providerOrder, err := s.provider.OrderDID(ctx, didww.OrderDIDRequest{
		SKUID: claimed.SKUID, AvailableDIDID: claimed.AvailableDIDID, ExternalReferenceID: claimed.ID.String(),
	})
	if err != nil {
		// Once the request starts, even a timeout or 4xx can hide a committed
		// provider order. Reconciliation, never resubmission, resolves it.
		unknown, updateErr := s.repo.MarkManagedOutcomeUnknown(ctx, claimed.ID, "provider_response_unknown", err.Error(), s.now().Add(managedReconcileDelay))
		if updateErr != nil {
			return ManagedOrder{}, apperror.NewInternal("persist uncertain provider outcome", updateErr)
		}
		return unknown, nil
	}
	if providerOrder.ID == "" || providerOrder.Attributes.ExternalReferenceID == nil || *providerOrder.Attributes.ExternalReferenceID != claimed.ID.String() {
		unknown, updateErr := s.repo.MarkManagedOutcomeUnknown(ctx, claimed.ID, "provider_identity_mismatch", "provider order identity could not be verified", s.now())
		if updateErr != nil {
			return ManagedOrder{}, apperror.NewInternal("persist uncertain provider outcome", updateErr)
		}
		return unknown, nil
	}
	return s.repo.RecordManagedProviderOrder(ctx, claimed.ID, providerOrder.ID, s.now())
}

func (s *Service) GetManagedOrder(ctx context.Context, organizationID, orderID uuid.UUID) (ManagedOrder, error) {
	order, err := s.repo.GetManagedOrder(ctx, organizationID, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ManagedOrder{}, apperror.NewNotFound("managed number order not found")
	}
	if err != nil {
		return ManagedOrder{}, apperror.NewInternal("get managed number order", err)
	}
	return order, nil
}

// Reconcile verifies provider order identity, DID ownership, inbound routing,
// and only then atomically creates and activates the customer phone number.
func (s *Service) Reconcile(ctx context.Context, orderID uuid.UUID) (ManagedOrder, error) {
	order, err := s.repo.GetManagedOrderInternal(ctx, orderID)
	if err != nil {
		return ManagedOrder{}, err
	}
	if order.Status == "completed" || order.Status == "failed" || order.Status == "manual_review" {
		return order, nil
	}

	var providerOrder didww.Order
	if order.ProviderOrderID != nil {
		providerOrder, err = s.provider.GetOrder(ctx, *order.ProviderOrderID)
	} else {
		var found bool
		providerOrder, found, err = s.provider.FindOrderByExternalReference(ctx, order.ID.String())
		if err == nil && !found {
			return s.repo.ScheduleManagedReconciliation(ctx, order.ID, "provider_order_not_visible", s.now().Add(managedReconcileDelay))
		}
	}
	if err != nil {
		return s.repo.ScheduleManagedReconciliation(ctx, order.ID, "provider_order_lookup_failed", s.now().Add(managedReconcileDelay))
	}
	if providerOrder.Attributes.ExternalReferenceID == nil || *providerOrder.Attributes.ExternalReferenceID != order.ID.String() {
		return s.repo.MarkManagedManualReview(ctx, order.ID, "provider_order_identity_mismatch")
	}
	if order.ProviderOrderID == nil {
		order, err = s.repo.RecordManagedProviderOrder(ctx, order.ID, providerOrder.ID, s.now())
		if err != nil {
			return ManagedOrder{}, err
		}
	}
	if providerOrder.Attributes.Status != "Completed" && providerOrder.Attributes.Status != "completed" {
		return s.repo.ScheduleManagedReconciliation(ctx, order.ID, "provider_order_pending", s.now().Add(managedReconcileDelay))
	}

	did, err := s.provider.FindDIDByNumber(ctx, order.Number)
	if err != nil {
		return s.repo.ScheduleManagedReconciliation(ctx, order.ID, "provider_did_not_visible", s.now().Add(managedReconcileDelay))
	}
	if did.ID == "" || normalizeProviderNumber(did.Attributes.Number) != order.Number || did.Attributes.Terminated {
		return s.repo.MarkManagedManualReview(ctx, order.ID, "provider_did_identity_mismatch")
	}
	order, err = s.repo.RecordManagedOwnedDID(ctx, order.ID, did.ID, s.now())
	if err != nil {
		return ManagedOrder{}, err
	}

	if !didUsesTrunk(did, s.inboundTrunkID) {
		if _, err = s.provider.AssignDIDVoiceInTrunk(ctx, did.ID, s.inboundTrunkID); err != nil {
			return s.repo.ScheduleManagedReconciliation(ctx, order.ID, "inbound_route_assignment_failed", s.now().Add(managedReconcileDelay))
		}
	}
	verified, err := s.provider.GetDID(ctx, did.ID)
	if err != nil || !didUsesTrunk(verified, s.inboundTrunkID) || normalizeProviderNumber(verified.Attributes.Number) != order.Number {
		return s.repo.ScheduleManagedReconciliation(ctx, order.ID, "inbound_route_not_verified", s.now().Add(managedReconcileDelay))
	}
	return s.activateManagedNumber(ctx, order.ID, did.ID, s.now())
}
func managedWriteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewConflict("managed number order changed concurrently")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.NewConflict("number already has a purchase in progress")
	}
	return apperror.NewInternal("persist managed number order", err)
}

func (s *Service) activateManagedNumber(ctx context.Context, orderID uuid.UUID, didID string, verifiedAt time.Time) (ManagedOrder, error) {
	if s.db == nil {
		return ManagedOrder{}, apperror.NewServiceUnavailable("managed number persistence is not configured", nil)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ManagedOrder{}, apperror.NewInternal("begin managed number activation", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repository := s.repo.WithQueries(sqlc.New(tx))
	order, err := repository.LockManagedOrder(ctx, orderID)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	if order.Status == "completed" {
		return order, tx.Commit(ctx)
	}
	if order.Status != "configuring" || order.ProviderDIDID == nil || *order.ProviderDIDID != didID {
		return ManagedOrder{}, apperror.NewConflict("managed number order is not ready for activation")
	}
	phoneNumber, err := repository.CreateActivatedManagedNumber(ctx, order, didID)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	order, err = repository.CompleteManagedOrder(ctx, order.ID, phoneNumber.ID, s.inboundTrunkID, verifiedAt)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ManagedOrder{}, apperror.NewInternal("commit managed number activation", err)
	}
	return order, nil
}

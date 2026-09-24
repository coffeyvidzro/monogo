package numbers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const managedReconcileDelay = 30 * time.Second

type ManagedPurchaseRequest struct {
	Number      string `json:"number"`
	CountryCode string `json:"country_code"`
}

type ManagedOrder struct {
	ID                     uuid.UUID  `json:"id"`
	OrganizationID         uuid.UUID  `json:"organization_id"`
	PhoneNumberID          *uuid.UUID `json:"phone_number_id,omitempty"`
	Number                 string     `json:"number"`
	CountryCode            string     `json:"country_code"`
	Status                 string     `json:"status"`
	SubmittedAt            *time.Time `json:"submitted_at,omitempty"`
	OwnershipVerifiedAt    *time.Time `json:"ownership_verified_at,omitempty"`
	RoutingVerifiedAt      *time.Time `json:"routing_verified_at,omitempty"`
	ActivatedAt            *time.Time `json:"activated_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	ProviderID             uuid.UUID  `json:"-"`
	AvailableDIDID         string     `json:"-"`
	SKUID                  string     `json:"-"`
	ProviderOrderID        *string    `json:"-"`
	ProviderDIDID          *string    `json:"-"`
	InboundTrunkID         *string    `json:"-"`
	IdempotencyKey         string     `json:"-"`
	RequestHash            string     `json:"-"`
	ReconcileAfter         time.Time  `json:"-"`
	ReconciliationAttempts int        `json:"-"`
}

type managedProvider interface {
	SearchAvailableDIDs(context.Context, didww.AvailableDIDFilter) (didww.AvailableDIDList, error)
	OrderDID(context.Context, didww.OrderDIDRequest) (didww.Order, error)
	GetOrder(context.Context, string) (didww.Order, error)
	FindOrderByExternalReference(context.Context, string) (didww.Order, bool, error)
	FindDIDByNumber(context.Context, string) (didww.DID, error)
	AssignDIDVoiceInTrunk(context.Context, string, string) (didww.DID, error)
	GetDID(context.Context, string) (didww.DID, error)
}

type ManagedService struct {
	repo           *ManagedRepository
	provider       managedProvider
	inboundTrunkID string
	now            func() time.Time
}

func NewManagedService(db *pgxpool.Pool, provider managedProvider, inboundTrunkID string) *ManagedService {
	return &ManagedService{repo: NewManagedRepository(db), provider: provider,
		inboundTrunkID: strings.TrimSpace(inboundTrunkID), now: time.Now}
}

func (s *ManagedService) Purchase(ctx context.Context, organizationID uuid.UUID, key string, req ManagedPurchaseRequest) (ManagedOrder, error) {
	key = strings.TrimSpace(key)
	if err := normalizeManagedPurchase(organizationID, key, &req); err != nil {
		return ManagedOrder{}, err
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

	digest := sha256.Sum256([]byte(req.Number + "\n" + req.CountryCode))
	order, created, err := s.repo.Create(ctx, organizationID, key, hex.EncodeToString(digest[:]), req, inventory.ID, skuID)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	if !created {
		if order.RequestHash != hex.EncodeToString(digest[:]) {
			return ManagedOrder{}, apperror.NewConflict("idempotency key was used with a different purchase")
		}
		return order, nil
	}
	return s.submit(ctx, order)
}

func (s *ManagedService) submit(ctx context.Context, order ManagedOrder) (ManagedOrder, error) {
	claimed, err := s.repo.ClaimSubmission(ctx, order.ID)
	if err != nil {
		return ManagedOrder{}, managedWriteError(err)
	}
	providerOrder, err := s.provider.OrderDID(ctx, didww.OrderDIDRequest{
		SKUID: claimed.SKUID, AvailableDIDID: claimed.AvailableDIDID, ExternalReferenceID: claimed.ID.String(),
	})
	if err != nil {
		// Once the request starts, even a timeout or 4xx can hide a committed
		// provider order. Reconciliation, never resubmission, resolves it.
		unknown, updateErr := s.repo.MarkOutcomeUnknown(ctx, claimed.ID, "provider_response_unknown", err.Error(), s.now().Add(managedReconcileDelay))
		if updateErr != nil {
			return ManagedOrder{}, apperror.NewInternal("persist uncertain provider outcome", updateErr)
		}
		return unknown, nil
	}
	if providerOrder.ID == "" || providerOrder.Attributes.ExternalReferenceID == nil || *providerOrder.Attributes.ExternalReferenceID != claimed.ID.String() {
		unknown, updateErr := s.repo.MarkOutcomeUnknown(ctx, claimed.ID, "provider_identity_mismatch", "provider order identity could not be verified", s.now())
		if updateErr != nil {
			return ManagedOrder{}, apperror.NewInternal("persist uncertain provider outcome", updateErr)
		}
		return unknown, nil
	}
	return s.repo.RecordProviderOrder(ctx, claimed.ID, providerOrder.ID, s.now())
}

func (s *ManagedService) Get(ctx context.Context, organizationID, orderID uuid.UUID) (ManagedOrder, error) {
	order, err := s.repo.Get(ctx, organizationID, orderID)
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
func (s *ManagedService) Reconcile(ctx context.Context, orderID uuid.UUID) (ManagedOrder, error) {
	order, err := s.repo.GetInternal(ctx, orderID)
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
			return s.repo.ScheduleReconciliation(ctx, order.ID, "provider_order_not_visible", s.now().Add(managedReconcileDelay))
		}
	}
	if err != nil {
		return s.repo.ScheduleReconciliation(ctx, order.ID, "provider_order_lookup_failed", s.now().Add(managedReconcileDelay))
	}
	if providerOrder.Attributes.ExternalReferenceID == nil || *providerOrder.Attributes.ExternalReferenceID != order.ID.String() {
		return s.repo.ManualReview(ctx, order.ID, "provider_order_identity_mismatch")
	}
	if order.ProviderOrderID == nil {
		order, err = s.repo.RecordProviderOrder(ctx, order.ID, providerOrder.ID, s.now())
		if err != nil {
			return ManagedOrder{}, err
		}
	}
	if providerOrder.Attributes.Status != "Completed" && providerOrder.Attributes.Status != "completed" {
		return s.repo.ScheduleReconciliation(ctx, order.ID, "provider_order_pending", s.now().Add(managedReconcileDelay))
	}

	did, err := s.provider.FindDIDByNumber(ctx, order.Number)
	if err != nil {
		return s.repo.ScheduleReconciliation(ctx, order.ID, "provider_did_not_visible", s.now().Add(managedReconcileDelay))
	}
	if did.ID == "" || normalizeProviderNumber(did.Attributes.Number) != order.Number || did.Attributes.Terminated {
		return s.repo.ManualReview(ctx, order.ID, "provider_did_identity_mismatch")
	}
	order, err = s.repo.RecordOwnedDID(ctx, order.ID, did.ID, s.now())
	if err != nil {
		return ManagedOrder{}, err
	}

	if !didUsesTrunk(did, s.inboundTrunkID) {
		if _, err = s.provider.AssignDIDVoiceInTrunk(ctx, did.ID, s.inboundTrunkID); err != nil {
			return s.repo.ScheduleReconciliation(ctx, order.ID, "inbound_route_assignment_failed", s.now().Add(managedReconcileDelay))
		}
	}
	verified, err := s.provider.GetDID(ctx, did.ID)
	if err != nil || !didUsesTrunk(verified, s.inboundTrunkID) || normalizeProviderNumber(verified.Attributes.Number) != order.Number {
		return s.repo.ScheduleReconciliation(ctx, order.ID, "inbound_route_not_verified", s.now().Add(managedReconcileDelay))
	}
	return s.repo.Activate(ctx, order.ID, did.ID, s.inboundTrunkID, s.now())
}

func normalizeManagedPurchase(org uuid.UUID, key string, req *ManagedPurchaseRequest) error {
	if org == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	if key == "" || len(key) > 255 {
		return apperror.NewBadRequest("Idempotency-Key is required and must not exceed 255 characters")
	}
	req.Number = strings.TrimSpace(req.Number)
	req.CountryCode = strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if !e164.MatchString(req.Number) {
		return apperror.NewBadRequest("number must be in E.164 format")
	}
	if len(req.CountryCode) != 2 || req.CountryCode[0] < 'A' || req.CountryCode[0] > 'Z' || req.CountryCode[1] < 'A' || req.CountryCode[1] > 'Z' {
		return apperror.NewBadRequest("country_code must be a two-letter ISO country code")
	}
	return nil
}

func normalizeProviderNumber(number string) string {
	number = strings.TrimSpace(number)
	if !strings.HasPrefix(number, "+") {
		number = "+" + number
	}
	return number
}

func availableDIDSKU(available didww.AvailableDID) (string, error) {
	for _, key := range []string{"sku", "did_group"} {
		raw, ok := available.Relationships[key]
		if !ok {
			continue
		}
		var relationship struct {
			Data *didww.ResourceIdentifier `json:"data"`
		}
		if json.Unmarshal(raw, &relationship) == nil && relationship.Data != nil && strings.TrimSpace(relationship.Data.ID) != "" {
			return relationship.Data.ID, nil
		}
	}
	return "", fmt.Errorf("available DID %q has no SKU relationship", available.ID)
}

func didUsesTrunk(did didww.DID, trunkID string) bool {
	rel, ok := did.Relationships["voice_in_trunk"]
	return ok && rel.Data != nil && rel.Data.Type == "voice_in_trunks" && rel.Data.ID == trunkID
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

package numbers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/modules/audit"
	"github.com/leamout/leamout/pkg/apperror"
)

type numberRepository interface {
	CreateBYOC(context.Context, uuid.UUID, CreateRequest) (sqlc.PhoneNumber, error)
	CreateManaged(context.Context, uuid.UUID, string, ManagedNumberPurchaseAuthorization) (sqlc.PhoneNumber, error)
	List(context.Context, uuid.UUID) ([]sqlc.PhoneNumber, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (sqlc.PhoneNumber, error)
	GetForRelease(context.Context, uuid.UUID, uuid.UUID) (sqlc.PhoneNumber, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateRequest) (sqlc.PhoneNumber, error)
	ReleaseBYOC(context.Context, uuid.UUID, uuid.UUID) (sqlc.PhoneNumber, error)
	SetCarrierConnection(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, audit.Event) (sqlc.PhoneNumber, error)
}

type providerOperationRepository interface {
	MarkProviderOperationAccepted(context.Context, sqlc.ProviderOperation, string, []byte) error
	RecordProviderOperationFailure(context.Context, uuid.UUID, error) error
	FailProviderOperation(context.Context, sqlc.ProviderOperation, error) error
	CompleteProviderOperation(context.Context, sqlc.ProviderOperation, ProviderOperationRequest, string, []byte) error
}

type ManagedNumberInventory interface {
	SearchAvailable(context.Context, AvailableSearchRequest) ([]ManagedNumberCandidate, error)
}

type ManagedNumberSelectionStore interface {
	SaveManagedSelection(context.Context, uuid.UUID, ManagedNumberCandidate) (string, error)
	LoadManagedSelection(context.Context, uuid.UUID, string) (ManagedNumberCandidate, error)
}

type ManagedNumberPurchaseAuthority interface {
	QuoteManagedNumberPurchase(context.Context, uuid.UUID) (uuid.UUID, int64, string, error)
	ReserveManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) (uuid.UUID, error)
	VerifyManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) error
	CaptureManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) error
	ReleaseManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

type ManagedNumberProvider interface {
	FindNumberOrder(context.Context, string) (ProviderOrder, bool, error)
	CreateNumberOrder(context.Context, ProviderOrderRequest) (ProviderOrder, error)
	FindManagedNumber(context.Context, string) (ProviderNumber, error)
	ConfigureNumberRouting(context.Context, string, string) (ProviderNumber, error)
}

type Service struct {
	repo              numberRepository
	providerRepo      providerOperationRepository
	managedInventory  ManagedNumberInventory
	managedSelections ManagedNumberSelectionStore
	managedPurchase   ManagedNumberPurchaseAuthority
	providers         map[string]ManagedNumberProvider
}

func NewService(repo numberRepository) *Service {
	s := &Service{repo: repo, providers: make(map[string]ManagedNumberProvider)}
	if providerRepo, ok := repo.(providerOperationRepository); ok {
		s.providerRepo = providerRepo
	}
	if store, ok := repo.(ManagedNumberSelectionStore); ok {
		s.managedSelections = store
	}
	return s
}

func (s *Service) SetManagedAcquisition(inventory ManagedNumberInventory) {
	s.managedInventory = inventory
}

func (s *Service) SetManagedSelectionStore(store ManagedNumberSelectionStore) {
	s.managedSelections = store
}

func (s *Service) SetManagedPurchaseAuthority(authority ManagedNumberPurchaseAuthority) {
	s.managedPurchase = authority
}

func (s *Service) SetManagedProvider(slug string, provider ManagedNumberProvider) {
	slug = strings.TrimSpace(slug)
	if slug == "" || provider == nil {
		return
	}
	s.providers[slug] = provider
}

func (s *Service) SearchAvailable(ctx context.Context, organizationID uuid.UUID, req AvailableSearchRequest) ([]AvailableNumberResponse, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}
	if s.managedInventory == nil || s.managedSelections == nil {
		return nil, apperror.NewServiceUnavailable("managed number inventory is not configured", nil)
	}
	if s.managedPurchase == nil {
		return nil, apperror.NewServiceUnavailable("managed number commercial authorization is not configured", nil)
	}
	countryCode, err := normalizeCountryCode(req.CountryCode)
	if err != nil {
		return nil, err
	}
	contains, err := normalizeNumberContains(req.Contains)
	if err != nil {
		return nil, err
	}
	req.CountryCode, req.Contains = countryCode, contains

	priceID, amountMinor, currency, err := s.managedPurchase.QuoteManagedNumberPurchase(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	candidates, err := s.managedInventory.SearchAvailable(ctx, req)
	if err != nil {
		return nil, apperror.NewServiceUnavailable("search managed number inventory", err)
	}
	result := make([]AvailableNumberResponse, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.Provider = strings.TrimSpace(candidate.Provider)
		candidate.ProviderInventoryID = strings.TrimSpace(candidate.ProviderInventoryID)
		candidate.ProviderProductID = strings.TrimSpace(candidate.ProviderProductID)
		if candidate.Provider == "" || candidate.ProviderInventoryID == "" || candidate.ProviderProductID == "" {
			return nil, apperror.NewServiceUnavailable("managed number provider returned incomplete purchase metadata", nil)
		}
		candidate.Number, err = normalizeNumber(candidate.Number)
		if err != nil {
			return nil, apperror.NewServiceUnavailable("managed number provider returned an invalid number", err)
		}
		candidate.CountryCode, err = normalizeCountryCode(candidate.CountryCode)
		if err != nil {
			return nil, apperror.NewServiceUnavailable("managed number provider returned an invalid country", err)
		}
		if candidate.ChannelsIncludedCount <= 0 {
			return nil, apperror.NewServiceUnavailable("managed number provider returned a number without included voice capacity", nil)
		}
		candidate.PriceID = priceID
		candidate.PriceAmountMinor = amountMinor
		candidate.PriceCurrency = currency
		selectionID, err := s.managedSelections.SaveManagedSelection(ctx, organizationID, candidate)
		if err != nil {
			return nil, apperror.NewServiceUnavailable("store managed number selection", err)
		}
		result = append(result, AvailableNumberResponse{
			SelectionID:  selectionID,
			Number:       candidate.Number,
			CountryCode:  candidate.CountryCode,
			VoiceEnabled: true,
			Price:        MoneyQuote{AmountMinor: amountMinor, Currency: currency},
		})
	}
	return result, nil
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.PhoneNumber, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	mode, err := normalizeProvisioningMode(req.Type)
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}
	req.Type = mode

	switch mode {
	case ProvisioningModeBYOC:
		if strings.TrimSpace(req.SelectionID) != "" {
			return sqlc.PhoneNumber{}, apperror.NewBadRequest("selection_id is only valid for managed numbers")
		}
		if req.CarrierConnectionID != nil && *req.CarrierConnectionID == uuid.Nil {
			return sqlc.PhoneNumber{}, apperror.NewBadRequest("carrier_connection_id must be valid when provided")
		}
		req.Number, err = normalizeNumber(req.Number)
		if err != nil {
			return sqlc.PhoneNumber{}, err
		}
		req.CountryCode, err = normalizeCountryCode(req.CountryCode)
		if err != nil {
			return sqlc.PhoneNumber{}, err
		}
		result, err := s.repo.CreateBYOC(ctx, organizationID, req)
		return result, writeError(err, "create BYOC phone number")

	case ProvisioningModeManaged:
		if strings.TrimSpace(req.Number) != "" || strings.TrimSpace(req.CountryCode) != "" || req.CarrierConnectionID != nil || req.VoiceEnabled != nil || req.SMSEnabled != nil {
			return sqlc.PhoneNumber{}, apperror.NewBadRequest("managed number creation accepts only type and selection_id")
		}
		if s.managedSelections == nil || s.managedPurchase == nil {
			return sqlc.PhoneNumber{}, apperror.NewServiceUnavailable("managed number purchasing is not configured", nil)
		}
		selectionID, err := normalizeSelectionID(req.SelectionID)
		if err != nil {
			return sqlc.PhoneNumber{}, err
		}
		selection, err := s.managedSelections.LoadManagedSelection(ctx, organizationID, selectionID)
		if err != nil {
			if errors.Is(err, ErrSelectionNotFound) {
				return sqlc.PhoneNumber{}, apperror.NewNotFound(err.Error())
			}
			return sqlc.PhoneNumber{}, apperror.NewServiceUnavailable("load managed number selection", err)
		}
		if selection.PriceID == uuid.Nil || selection.PriceAmountMinor <= 0 || strings.TrimSpace(selection.PriceCurrency) == "" {
			return sqlc.PhoneNumber{}, apperror.NewConflict("managed number selection does not contain a valid quote")
		}

		authorizationID := uuid.New()
		reservationID, err := s.managedPurchase.ReserveManagedNumberPurchase(
			ctx,
			organizationID,
			authorizationID,
			selection.PriceID,
			selection.PriceAmountMinor,
			selection.PriceCurrency,
		)
		if err != nil {
			return sqlc.PhoneNumber{}, err
		}
		authorization := ManagedNumberPurchaseAuthorization{
			ID:            authorizationID,
			ReservationID: reservationID,
			PriceID:       selection.PriceID,
			AmountMinor:   selection.PriceAmountMinor,
			Currency:      selection.PriceCurrency,
		}
		result, err := s.repo.CreateManaged(ctx, organizationID, selectionID, authorization)
		if err != nil {
			if releaseErr := s.managedPurchase.ReleaseManagedNumberPurchase(ctx, organizationID, authorizationID, reservationID); releaseErr != nil {
				return sqlc.PhoneNumber{}, apperror.NewServiceUnavailable("release managed number purchase authorization", releaseErr)
			}
			switch {
			case errors.Is(err, ErrSelectionNotFound):
				return sqlc.PhoneNumber{}, apperror.NewNotFound(err.Error())
			case errors.Is(err, ErrSelectionUnavailable):
				return sqlc.PhoneNumber{}, apperror.NewConflict(err.Error())
			case errors.Is(err, ErrProviderRoutingUnavailable):
				return sqlc.PhoneNumber{}, apperror.NewServiceUnavailable(err.Error(), err)
			}
			if errors.Is(err, pgx.ErrNoRows) {
				return sqlc.PhoneNumber{}, apperror.NewNotFound("active organization or managed provider not found")
			}
			return sqlc.PhoneNumber{}, writeError(err, "create managed phone number")
		}
		return result, nil
	}
	return sqlc.PhoneNumber{}, apperror.NewBadRequest("unsupported number type")
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.PhoneNumber, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}
	result, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list phone numbers", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	result, err := s.repo.Get(ctx, organizationID, id)
	return result, readError(err, "phone number not found")
}

func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.PhoneNumber, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if req.VoiceEnabled == nil && req.SMSEnabled == nil {
		return sqlc.PhoneNumber{}, apperror.NewBadRequest("at least one field is required")
	}
	result, err := s.repo.Update(ctx, organizationID, id, req)
	return result, writeError(err, "phone number not found")
}

func (s *Service) Release(ctx context.Context, organizationID, id uuid.UUID) error {
	if err := validIDs(organizationID, id); err != nil {
		return err
	}
	current, err := s.repo.GetForRelease(ctx, organizationID, id)
	if err != nil {
		return readError(err, "phone number not found")
	}
	if current.ProvisioningMode != string(ProvisioningModeBYOC) {
		return apperror.NewConflict("managed phone number release is not available yet")
	}
	_, err = s.repo.ReleaseBYOC(ctx, organizationID, id)
	return writeError(err, "release phone number")
}

func (s *Service) SetCarrierConnection(ctx context.Context, organizationID, id uuid.UUID, req CarrierConnectionRequest) (sqlc.PhoneNumber, error) {
	if err := validIDs(organizationID, id); err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if req.CarrierConnectionID == uuid.Nil {
		return sqlc.PhoneNumber{}, apperror.NewBadRequest("carrier_connection_id is required")
	}
	current, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return sqlc.PhoneNumber{}, err
	}
	if current.ProvisioningMode != string(ProvisioningModeBYOC) {
		return sqlc.PhoneNumber{}, apperror.NewConflict("managed phone number carrier connections are platform-managed")
	}
	actor, err := audit.ActorFromContext(ctx)
	if err != nil {
		return sqlc.PhoneNumber{}, apperror.NewInternal("attribute number assignment audit event", err)
	}
	action := "number.carrier_assigned"
	metadata := map[string]any{"carrier_connection_id": req.CarrierConnectionID}
	if current.CarrierConnectionID != nil {
		action = "number.carrier_reassigned"
		metadata["previous_carrier_connection_id"] = *current.CarrierConnectionID
	}
	event, err := audit.NewEvent(organizationID, actor, action, "phone_number", id, metadata)
	if err != nil {
		return sqlc.PhoneNumber{}, apperror.NewInternal("create number assignment audit event", err)
	}
	result, err := s.repo.SetCarrierConnection(ctx, organizationID, id, req.CarrierConnectionID, event)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.PhoneNumber{}, apperror.NewNotFound("phone number or active carrier connection not found")
	}
	return result, writeError(err, "assign carrier connection")
}

func (s *Service) ExecuteProviderOperation(ctx context.Context, operation sqlc.ProviderOperation) error {
	if operation.OperationType != "number_provision" {
		return nil
	}
	if s.providerRepo == nil {
		return fmt.Errorf("provider operation repository is not configured")
	}
	var request ProviderOperationRequest
	if err := json.Unmarshal(operation.Request, &request); err != nil {
		return s.failOperation(ctx, operation, request, fmt.Errorf("decode provider operation request: %w", err))
	}
	if err := validateProviderOperationRequest(request); err != nil {
		return s.failOperation(ctx, operation, request, err)
	}
	if s.managedPurchase == nil {
		return s.failOperation(ctx, operation, request, fmt.Errorf("managed number commercial authorization is not configured"))
	}
	authorization := request.PurchaseAuthorization
	if err := s.managedPurchase.VerifyManagedNumberPurchase(
		ctx,
		operation.OrganizationID,
		authorization.ID,
		authorization.ReservationID,
		authorization.PriceID,
		authorization.AmountMinor,
		authorization.Currency,
	); err != nil {
		return s.failOperation(ctx, operation, request, fmt.Errorf("verify managed number purchase authorization: %w", err))
	}

	provider := s.providers[request.Provider]
	if provider == nil {
		err := fmt.Errorf("managed number provider %q is not configured", request.Provider)
		if recordErr := s.providerRepo.RecordProviderOperationFailure(ctx, operation.ID, err); recordErr != nil {
			return fmt.Errorf("record provider configuration failure: %w", recordErr)
		}
		return nil
	}

	externalReferenceID := operation.ID.String()
	providerOrder, found, err := provider.FindNumberOrder(ctx, externalReferenceID)
	if err != nil {
		return s.handleProviderError(ctx, operation, request, fmt.Errorf("reconcile provider number order: %w", err))
	}
	if !found {
		providerOrder, err = provider.CreateNumberOrder(ctx, ProviderOrderRequest{
			ProviderInventoryID: request.ProviderInventoryID,
			ProviderProductID:   request.ProviderProductID,
			ExternalReferenceID: externalReferenceID,
		})
		if err != nil {
			return s.handleProviderError(ctx, operation, request, fmt.Errorf("create provider number order: %w", err))
		}
	}
	providerResponse, err := json.Marshal(providerOrder)
	if err != nil {
		return fmt.Errorf("encode provider number order response: %w", err)
	}
	if err := s.providerRepo.MarkProviderOperationAccepted(ctx, operation, providerOrder.ID, providerResponse); err != nil {
		return fmt.Errorf("persist accepted provider number order: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(providerOrder.Status)) {
	case "pending":
		return nil
	case "canceled", "cancelled", "failed":
		return s.failOperation(ctx, operation, request, fmt.Errorf("provider number order ended with status %q", providerOrder.Status))
	case "completed":
		return s.completeProviderOperation(ctx, operation, request, provider, providerResponse)
	default:
		err := fmt.Errorf("provider number order returned unknown status %q", providerOrder.Status)
		if recordErr := s.providerRepo.RecordProviderOperationFailure(ctx, operation.ID, err); recordErr != nil {
			return fmt.Errorf("record unknown provider order status: %w", recordErr)
		}
		return nil
	}
}

func (s *Service) completeProviderOperation(ctx context.Context, operation sqlc.ProviderOperation, request ProviderOperationRequest, provider ManagedNumberProvider, providerResponse []byte) error {
	authorization := request.PurchaseAuthorization
	if err := s.managedPurchase.CaptureManagedNumberPurchase(
		ctx,
		operation.OrganizationID,
		authorization.ID,
		authorization.ReservationID,
		authorization.PriceID,
		authorization.AmountMinor,
		authorization.Currency,
	); err != nil {
		return fmt.Errorf("capture completed managed number provider order: %w", err)
	}

	providerNumber, err := provider.FindManagedNumber(ctx, request.Number)
	if err != nil {
		return s.handleCapturedProviderError(ctx, operation, fmt.Errorf("resolve purchased provider number: %w", err))
	}
	if strings.TrimSpace(providerNumber.ID) == "" {
		return s.handleCapturedProviderError(ctx, operation, fmt.Errorf("provider returned purchased number without resource id"))
	}
	if strings.TrimSpace(providerNumber.Number) != request.Number {
		return s.failProviderOperation(ctx, operation, fmt.Errorf("provider returned unexpected purchased number %q", providerNumber.Number))
	}
	if strings.TrimSpace(providerNumber.RoutingResourceID) != request.ProviderRoutingResourceID {
		providerNumber, err = provider.ConfigureNumberRouting(ctx, providerNumber.ID, request.ProviderRoutingResourceID)
		if err != nil {
			return s.handleCapturedProviderError(ctx, operation, fmt.Errorf("configure provider number routing: %w", err))
		}
		if strings.TrimSpace(providerNumber.RoutingResourceID) != request.ProviderRoutingResourceID {
			return s.handleCapturedProviderError(ctx, operation, fmt.Errorf("provider number routing did not converge to expected resource"))
		}
	}
	if err := s.providerRepo.CompleteProviderOperation(ctx, operation, request, providerNumber.ID, providerResponse); err != nil {
		return fmt.Errorf("complete managed number provider operation: %w", err)
	}
	return nil
}

func (s *Service) handleProviderError(ctx context.Context, operation sqlc.ProviderOperation, request ProviderOperationRequest, err error) error {
	var classified interface{ Retryable() bool }
	if errors.As(err, &classified) && !classified.Retryable() {
		return s.failOperation(ctx, operation, request, err)
	}
	if recordErr := s.providerRepo.RecordProviderOperationFailure(ctx, operation.ID, err); recordErr != nil {
		return fmt.Errorf("record provider operation failure: %w", recordErr)
	}
	return nil
}

func (s *Service) handleCapturedProviderError(ctx context.Context, operation sqlc.ProviderOperation, err error) error {
	var classified interface{ Retryable() bool }
	if errors.As(err, &classified) && !classified.Retryable() {
		return s.failProviderOperation(ctx, operation, err)
	}
	if recordErr := s.providerRepo.RecordProviderOperationFailure(ctx, operation.ID, err); recordErr != nil {
		return fmt.Errorf("record provider operation failure after capture: %w", recordErr)
	}
	return nil
}

func (s *Service) failOperation(ctx context.Context, operation sqlc.ProviderOperation, request ProviderOperationRequest, err error) error {
	authorization := request.PurchaseAuthorization
	if s.managedPurchase != nil && authorization.ID != uuid.Nil && authorization.ReservationID != uuid.Nil {
		if releaseErr := s.managedPurchase.ReleaseManagedNumberPurchase(
			ctx,
			operation.OrganizationID,
			authorization.ID,
			authorization.ReservationID,
		); releaseErr != nil {
			return fmt.Errorf("release managed number purchase authorization: %w", releaseErr)
		}
	}
	return s.failProviderOperation(ctx, operation, err)
}

func (s *Service) failProviderOperation(ctx context.Context, operation sqlc.ProviderOperation, err error) error {
	if failErr := s.providerRepo.FailProviderOperation(ctx, operation, err); failErr != nil {
		return fmt.Errorf("fail provider operation: %w", failErr)
	}
	return nil
}

func validateProviderOperationRequest(request ProviderOperationRequest) error {
	authorization := request.PurchaseAuthorization
	if strings.TrimSpace(request.Provider) == "" || strings.TrimSpace(request.ProviderInventoryID) == "" ||
		strings.TrimSpace(request.ProviderProductID) == "" || strings.TrimSpace(request.Number) == "" ||
		strings.TrimSpace(request.CountryCode) == "" || request.CarrierConnectionID == uuid.Nil ||
		strings.TrimSpace(request.ProviderRoutingResourceID) == "" || authorization.ID == uuid.Nil ||
		authorization.ReservationID == uuid.Nil || authorization.PriceID == uuid.Nil || authorization.AmountMinor <= 0 ||
		strings.TrimSpace(authorization.Currency) == "" {
		return fmt.Errorf("provider operation request is incomplete")
	}
	return nil
}

func validIDs(organizationID, id uuid.UUID) error {
	if err := validateOrganizationID(organizationID); err != nil {
		return err
	}
	return validateNumberID(id)
}

func readError(err error, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}

func writeError(err error, message string) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.NewConflict("phone number already exists")
	}
	return readError(err, message)
}

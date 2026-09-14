package numbers

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/modules/audit"
)

type fakeNumberRepository struct {
	createdBYOC          CreateRequest
	managedSelectionID   string
	managedAuthorization ManagedNumberPurchaseAuthorization
	managedCreateErr     error
	getNumber            sqlc.PhoneNumber
	getForRelease        sqlc.PhoneNumber
	releaseCalls         int
	selectionOrgID       uuid.UUID
	selectionCandidate   ManagedNumberCandidate
	selectionID          string
}

func (f *fakeNumberRepository) CreateBYOC(_ context.Context, _ uuid.UUID, req CreateRequest) (sqlc.PhoneNumber, error) {
	f.createdBYOC = req
	return sqlc.PhoneNumber{Number: req.Number, CountryCode: req.CountryCode, ProvisioningMode: string(ProvisioningModeBYOC), Status: "active"}, nil
}
func (f *fakeNumberRepository) CreateManaged(_ context.Context, _ uuid.UUID, selectionID string, authorization ManagedNumberPurchaseAuthorization) (sqlc.PhoneNumber, error) {
	f.managedSelectionID = selectionID
	f.managedAuthorization = authorization
	if f.managedCreateErr != nil {
		return sqlc.PhoneNumber{}, f.managedCreateErr
	}
	return sqlc.PhoneNumber{ProvisioningMode: string(ProvisioningModeManaged), Status: "provisioning"}, nil
}
func (f *fakeNumberRepository) List(context.Context, uuid.UUID) ([]sqlc.PhoneNumber, error) {
	return nil, nil
}
func (f *fakeNumberRepository) Get(context.Context, uuid.UUID, uuid.UUID) (sqlc.PhoneNumber, error) {
	return f.getNumber, nil
}
func (f *fakeNumberRepository) GetForRelease(context.Context, uuid.UUID, uuid.UUID) (sqlc.PhoneNumber, error) {
	return f.getForRelease, nil
}
func (f *fakeNumberRepository) Update(context.Context, uuid.UUID, uuid.UUID, UpdateRequest) (sqlc.PhoneNumber, error) {
	return f.getNumber, nil
}
func (f *fakeNumberRepository) ReleaseBYOC(context.Context, uuid.UUID, uuid.UUID) (sqlc.PhoneNumber, error) {
	f.releaseCalls++
	return f.getForRelease, nil
}
func (f *fakeNumberRepository) SetCarrierConnection(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, audit.Event) (sqlc.PhoneNumber, error) {
	return f.getNumber, nil
}
func (f *fakeNumberRepository) SaveManagedSelection(_ context.Context, organizationID uuid.UUID, candidate ManagedNumberCandidate) (string, error) {
	f.selectionOrgID = organizationID
	f.selectionCandidate = candidate
	return f.selectionID, nil
}
func (f *fakeNumberRepository) LoadManagedSelection(_ context.Context, organizationID uuid.UUID, selectionID string) (ManagedNumberCandidate, error) {
	f.selectionOrgID = organizationID
	if selectionID != f.selectionID && f.selectionID != "" {
		return ManagedNumberCandidate{}, ErrSelectionNotFound
	}
	return f.selectionCandidate, nil
}

type fakeManagedInventory struct {
	request    AvailableSearchRequest
	candidates []ManagedNumberCandidate
}

func (f *fakeManagedInventory) SearchAvailable(_ context.Context, request AvailableSearchRequest) ([]ManagedNumberCandidate, error) {
	f.request = request
	return f.candidates, nil
}

type fakeManagedPurchaseAuthority struct {
	priceID       uuid.UUID
	amountMinor   int64
	currency      string
	reservationID uuid.UUID
	reserveErr    error
	verifyErr     error
	captureErr    error
	releaseErr    error
	reserves      int
	verifies      int
	captures      int
	releases      int
}

func (f *fakeManagedPurchaseAuthority) QuoteManagedNumberPurchase(context.Context, uuid.UUID) (uuid.UUID, int64, string, error) {
	return f.priceID, f.amountMinor, f.currency, nil
}
func (f *fakeManagedPurchaseAuthority) ReserveManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) (uuid.UUID, error) {
	f.reserves++
	return f.reservationID, f.reserveErr
}
func (f *fakeManagedPurchaseAuthority) VerifyManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) error {
	f.verifies++
	return f.verifyErr
}
func (f *fakeManagedPurchaseAuthority) CaptureManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) error {
	f.captures++
	return f.captureErr
}
func (f *fakeManagedPurchaseAuthority) ReleaseManagedNumberPurchase(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	f.releases++
	return f.releaseErr
}

func TestCreateDispatchesBYOC(t *testing.T) {
	repo := &fakeNumberRepository{}
	service := NewService(repo)
	_, err := service.Create(context.Background(), uuid.New(), CreateRequest{Type: ProvisioningModeBYOC, Number: " +233201234567 ", CountryCode: " gh "})
	if err != nil {
		t.Fatal(err)
	}
	if repo.createdBYOC.Number != "+233201234567" || repo.createdBYOC.CountryCode != "GH" {
		t.Fatalf("created=%+v", repo.createdBYOC)
	}
}

func TestCreateDispatchesManagedSelection(t *testing.T) {
	priceID := uuid.New()
	reservationID := uuid.New()
	repo := &fakeNumberRepository{
		selectionID: "sel_test",
		selectionCandidate: ManagedNumberCandidate{
			Provider: "didww", ProviderInventoryID: "available-1", ProviderProductID: "sku-1",
			Number: "+15551236001", CountryCode: "US", ChannelsIncludedCount: 2,
			PriceID: priceID, PriceAmountMinor: 2500, PriceCurrency: "USD",
		},
	}
	authority := &fakeManagedPurchaseAuthority{priceID: priceID, amountMinor: 2500, currency: "USD", reservationID: reservationID}
	service := NewService(repo)
	service.SetManagedPurchaseAuthority(authority)
	result, err := service.Create(context.Background(), uuid.New(), CreateRequest{Type: ProvisioningModeManaged, SelectionID: " sel_test "})
	if err != nil {
		t.Fatal(err)
	}
	if repo.managedSelectionID != "sel_test" {
		t.Fatalf("selection=%q", repo.managedSelectionID)
	}
	if repo.managedAuthorization.ReservationID != reservationID || repo.managedAuthorization.PriceID != priceID {
		t.Fatalf("authorization=%+v", repo.managedAuthorization)
	}
	if authority.reserves != 1 || authority.releases != 0 {
		t.Fatalf("reserve/release=%d/%d, want 1/0", authority.reserves, authority.releases)
	}
	if result.Status != "provisioning" {
		t.Fatalf("status=%q", result.Status)
	}
}

func TestManagedCreateReleasesReservationWhenPersistenceFails(t *testing.T) {
	priceID := uuid.New()
	repo := &fakeNumberRepository{
		selectionID: "sel_test",
		selectionCandidate: ManagedNumberCandidate{
			Provider: "didww", ProviderInventoryID: "available-1", ProviderProductID: "sku-1",
			Number: "+15551236001", CountryCode: "US", ChannelsIncludedCount: 2,
			PriceID: priceID, PriceAmountMinor: 2500, PriceCurrency: "USD",
		},
		managedCreateErr: errors.New("database unavailable"),
	}
	authority := &fakeManagedPurchaseAuthority{reservationID: uuid.New()}
	service := NewService(repo)
	service.SetManagedPurchaseAuthority(authority)
	_, err := service.Create(context.Background(), uuid.New(), CreateRequest{Type: ProvisioningModeManaged, SelectionID: "sel_test"})
	if err == nil {
		t.Fatal("Create() unexpectedly succeeded")
	}
	if authority.reserves != 1 || authority.releases != 1 {
		t.Fatalf("reserve/release=%d/%d, want 1/1", authority.reserves, authority.releases)
	}
}

func TestManagedCreateRejectsBYOCFields(t *testing.T) {
	service := NewService(&fakeNumberRepository{})
	_, err := service.Create(context.Background(), uuid.New(), CreateRequest{Type: ProvisioningModeManaged, SelectionID: "sel_test", Number: "+233201234567"})
	if err == nil {
		t.Fatal("managed create accepted caller-supplied number")
	}
}

func TestSearchAvailableReturnsOpaqueCustomerResponse(t *testing.T) {
	priceID := uuid.New()
	repo := &fakeNumberRepository{selectionID: "sel_test"}
	inventory := &fakeManagedInventory{candidates: []ManagedNumberCandidate{{Provider: "didww", ProviderInventoryID: "available-1", ProviderProductID: "sku-1", Number: "+12125550100", CountryCode: "US", ChannelsIncludedCount: 2}}}
	authority := &fakeManagedPurchaseAuthority{priceID: priceID, amountMinor: 2500, currency: "USD"}
	service := NewService(repo)
	service.SetManagedAcquisition(inventory)
	service.SetManagedPurchaseAuthority(authority)
	result, err := service.SearchAvailable(context.Background(), uuid.New(), AvailableSearchRequest{CountryCode: " us ", Contains: "+212"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].SelectionID != "sel_test" {
		t.Fatalf("result=%+v", result)
	}
	if result[0].Price.AmountMinor != 2500 || result[0].Price.Currency != "USD" {
		t.Fatalf("price=%+v", result[0].Price)
	}
	if repo.selectionCandidate.PriceID != priceID || repo.selectionCandidate.PriceAmountMinor != 2500 || repo.selectionCandidate.PriceCurrency != "USD" {
		t.Fatalf("stored selection quote=%+v", repo.selectionCandidate)
	}
}

func TestReleaseRejectsManagedNumber(t *testing.T) {
	repo := &fakeNumberRepository{getForRelease: sqlc.PhoneNumber{ProvisioningMode: string(ProvisioningModeManaged), Status: "active"}}
	service := NewService(repo)
	if err := service.Release(context.Background(), uuid.New(), uuid.New()); err == nil {
		t.Fatal("release accepted managed number")
	}
	if repo.releaseCalls != 0 {
		t.Fatalf("release calls=%d", repo.releaseCalls)
	}
}

func TestResponseUsesTypeAndHidesManagedCarrier(t *testing.T) {
	connectionID := uuid.New()
	managed := response(sqlc.PhoneNumber{ProvisioningMode: string(ProvisioningModeManaged), CarrierConnectionID: &connectionID})
	if managed.Type != ProvisioningModeManaged {
		t.Fatalf("type=%q", managed.Type)
	}
	if managed.CarrierConnectionID != nil {
		t.Fatal("managed response exposed platform carrier")
	}
	byoc := response(sqlc.PhoneNumber{ProvisioningMode: string(ProvisioningModeBYOC), CarrierConnectionID: &connectionID})
	if byoc.Type != ProvisioningModeBYOC || byoc.CarrierConnectionID == nil {
		t.Fatalf("response=%+v", byoc)
	}
}

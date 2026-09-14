package trunks

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/modules/outbox"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/hasher"
)

type Service struct {
	repo   *Repository
	db     *pgxpool.Pool
	outbox *outbox.Repository
}

func NewService(repo *Repository, db ...*pgxpool.Pool) *Service {
	service := &Service{repo: repo}
	if len(db) > 0 && db[0] != nil {
		service.db = db[0]
		service.outbox = outbox.NewRepository(sqlc.New(db[0]))
	}
	return service
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (CreateResult, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return CreateResult{}, err
	}
	mode, err := normalizeProvisioningMode(req.Type)
	if err != nil {
		return CreateResult{}, err
	}
	req.Type = mode
	name, err := normalizeName(req.Name)
	if err != nil {
		return CreateResult{}, err
	}
	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil {
			return CreateResult{}, err
		}
		req.Direction = &value
	}
	if req.Status != nil {
		value, err := normalizeChoice(*req.Status, statuses, "status")
		if err != nil {
			return CreateResult{}, err
		}
		req.Status = &value
	}
	switch mode {
	case ProvisioningModeManaged:
		if req.CarrierConnectionID != nil {
			return CreateResult{}, apperror.NewBadRequest("carrier_connection_id is not accepted for managed trunks")
		}
		credential, ha1, err := s.newManagedSIPCredential()
		if err != nil {
			return CreateResult{}, apperror.NewInternal("generate managed SIP credential", err)
		}
		item, err := s.mutateTrunk(ctx, EventTrunkCreated, func(repo *Repository) (sqlc.Trunk, error) {
			trunk, err := repo.CreateManaged(ctx, organizationID, name, req.Direction, req.Status)
			if err != nil {
				return sqlc.Trunk{}, err
			}
			if _, err := repo.CreateCredential(ctx, organizationID, trunk.ID, credential.Username, credential.Realm, ha1); err != nil {
				return sqlc.Trunk{}, err
			}
			return trunk, nil
		})
		if err != nil {
			return CreateResult{}, writeError(err, "managed trunk", "managed trunk could not be created")
		}
		return CreateResult{Trunk: item, Credential: &credential}, nil
	case ProvisioningModeBYOC:
		if req.CarrierConnectionID == nil {
			return CreateResult{}, apperror.NewBadRequest("carrier_connection_id is required for BYOC trunks")
		}
		if err := validateID(*req.CarrierConnectionID, "carrier_connection_id"); err != nil {
			return CreateResult{}, err
		}
		item, err := s.mutateTrunk(ctx, EventTrunkCreated, func(repo *Repository) (sqlc.Trunk, error) {
			return repo.Create(ctx, sqlc.CreateTrunkParams{OrganizationID: &organizationID, CarrierConnectionID: *req.CarrierConnectionID, ProvisioningMode: string(ProvisioningModeBYOC), Name: name, Direction: req.Direction, Status: req.Status})
		})
		if err != nil {
			return CreateResult{}, writeError(err, "trunk", "carrier connection not found")
		}
		return CreateResult{Trunk: item}, nil
	}
	return CreateResult{}, apperror.NewBadRequest("type must be byoc or managed")
}

func (s *Service) RotateCredential(ctx context.Context, organizationID, trunkID uuid.UUID) (SIPCredential, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return SIPCredential{}, err
	}
	if err := validateID(trunkID, "trunk id"); err != nil {
		return SIPCredential{}, err
	}
	item, err := s.Get(ctx, organizationID, trunkID)
	if err != nil {
		return SIPCredential{}, err
	}
	if ProvisioningMode(item.ProvisioningMode) != ProvisioningModeManaged || item.CarrierConnectionID != nil {
		return SIPCredential{}, apperror.NewConflict("SIP credentials are only available for Cloud-authoritative managed trunks")
	}
	if item.Status != "active" {
		return SIPCredential{}, apperror.NewConflict("managed trunk must be active before rotating SIP credentials")
	}
	credential, ha1, err := s.newManagedSIPCredential()
	if err != nil {
		return SIPCredential{}, apperror.NewInternal("generate managed SIP credential", err)
	}
	if err := s.rotateCredential(ctx, item, credential, ha1); err != nil {
		return SIPCredential{}, err
	}
	return credential, nil
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.Trunk, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return nil, err
	}
	items, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list trunks", err)
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Trunk, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return sqlc.Trunk{}, err
	}
	if err := validateID(id, "trunk id"); err != nil {
		return sqlc.Trunk{}, err
	}
	item, err := s.repo.Get(ctx, organizationID, id)
	return item, readError(err, "trunk not found")
}

func (s *Service) Update(ctx context.Context, organizationID uuid.UUID, id uuid.UUID, req UpdateRequest) (sqlc.Trunk, error) {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return sqlc.Trunk{}, err
	}
	if req.Name == nil && req.Direction == nil && req.Status == nil {
		return sqlc.Trunk{}, apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil {
		value, err := normalizeName(*req.Name)
		if err != nil {
			return sqlc.Trunk{}, err
		}
		req.Name = &value
	}
	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil {
			return sqlc.Trunk{}, err
		}
		req.Direction = &value
	}
	if req.Status != nil {
		value, err := normalizeChoice(*req.Status, statuses, "status")
		if err != nil {
			return sqlc.Trunk{}, err
		}
		req.Status = &value
	}
	item, err := s.mutateTrunk(ctx, EventTrunkUpdated, func(repo *Repository) (sqlc.Trunk, error) {
		return repo.Update(ctx, sqlc.UpdateTrunkParams{Name: req.Name, Direction: req.Direction, Status: req.Status, ID: id, OrganizationID: &organizationID})
	})
	return item, writeError(err, "trunk", "trunk not found")
}

func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	item, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if item.Status == "disabled" {
		return nil
	}
	_, err = s.mutateTrunk(ctx, EventTrunkDisabled, func(repo *Repository) (sqlc.Trunk, error) {
		return repo.Disable(ctx, organizationID, id)
	})
	return writeError(err, "trunk", "trunk not found")
}

func (s *Service) CreateEndpoint(ctx context.Context, organizationID, trunkID uuid.UUID, req EndpointCreateRequest) (sqlc.TrunkEndpoint, error) {
	if _, err := s.requireBYOCTrunk(ctx, organizationID, trunkID); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	host, err := normalizeHost(req.Host)
	if err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	if req.Port != nil {
		if err := validatePort(*req.Port); err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
	}
	if req.Transport != nil {
		value, err := normalizeChoice(*req.Transport, transports, "transport")
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Transport = &value
	}
	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Direction = &value
	}
	if req.Priority != nil {
		if err := validatePriority(*req.Priority); err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
	}
	if req.Weight != nil {
		if err := validateWeight(*req.Weight); err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
	}
	item, err := s.mutateEndpoint(ctx, EventTrunkEndpointCreated, func(repo *Repository) (sqlc.TrunkEndpoint, error) {
		return repo.CreateEndpoint(ctx, sqlc.CreateTrunkEndpointParams{OrganizationID: &organizationID, TrunkID: trunkID, Host: host, Port: req.Port, Transport: req.Transport, Direction: req.Direction, Priority: req.Priority, Weight: req.Weight, Enabled: req.Enabled})
	})
	return item, writeError(err, "trunk endpoint", "trunk not found")
}

func (s *Service) ListEndpoints(ctx context.Context, organizationID, trunkID uuid.UUID) ([]sqlc.TrunkEndpoint, error) {
	if _, err := s.requireBYOCTrunk(ctx, organizationID, trunkID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListEndpoints(ctx, organizationID, trunkID)
	if err != nil {
		return nil, apperror.NewInternal("list trunk endpoints", err)
	}
	return items, nil
}

func (s *Service) GetEndpoint(ctx context.Context, organizationID, trunkID, id uuid.UUID) (sqlc.TrunkEndpoint, error) {
	if _, err := s.requireBYOCTrunk(ctx, organizationID, trunkID); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	if err := validateID(id, "endpoint id"); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	item, err := s.repo.GetEndpoint(ctx, organizationID, trunkID, id)
	return item, readError(err, "trunk endpoint not found")
}

func (s *Service) UpdateEndpoint(ctx context.Context, organizationID, trunkID, id uuid.UUID, req EndpointUpdateRequest) (sqlc.TrunkEndpoint, error) {
	if _, err := s.GetEndpoint(ctx, organizationID, trunkID, id); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	if req.Host == nil && req.Port == nil && req.Transport == nil && req.Direction == nil && req.Priority == nil && req.Weight == nil && req.Enabled == nil {
		return sqlc.TrunkEndpoint{}, apperror.NewBadRequest("at least one field is required")
	}
	if req.Host != nil {
		value, err := normalizeHost(*req.Host)
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Host = &value
	}
	if req.Port != nil {
		if err := validatePort(*req.Port); err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
	}
	if req.Transport != nil {
		value, err := normalizeChoice(*req.Transport, transports, "transport")
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Transport = &value
	}
	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Direction = &value
	}
	if req.Priority != nil {
		if err := validatePriority(*req.Priority); err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
	}
	if req.Weight != nil {
		if err := validateWeight(*req.Weight); err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
	}
	item, err := s.mutateEndpoint(ctx, EventTrunkEndpointUpdated, func(repo *Repository) (sqlc.TrunkEndpoint, error) {
		return repo.UpdateEndpoint(ctx, sqlc.UpdateTrunkEndpointParams{Host: req.Host, Port: req.Port, Transport: req.Transport, Direction: req.Direction, Priority: req.Priority, Weight: req.Weight, Enabled: req.Enabled, ID: id, TrunkID: trunkID, OrganizationID: &organizationID})
	})
	return item, writeError(err, "trunk endpoint", "trunk endpoint not found")
}

func (s *Service) DeleteEndpoint(ctx context.Context, organizationID, trunkID, id uuid.UUID) error {
	if _, err := s.GetEndpoint(ctx, organizationID, trunkID, id); err != nil {
		return err
	}
	_, err := s.mutateEndpoint(ctx, EventTrunkEndpointDeleted, func(repo *Repository) (sqlc.TrunkEndpoint, error) {
		return repo.DeleteEndpoint(ctx, organizationID, trunkID, id)
	})
	return writeError(err, "trunk endpoint", "trunk endpoint not found")
}

func (s *Service) requireBYOCTrunk(ctx context.Context, organizationID, trunkID uuid.UUID) (sqlc.Trunk, error) {
	item, err := s.Get(ctx, organizationID, trunkID)
	if err != nil {
		return sqlc.Trunk{}, err
	}
	if ProvisioningMode(item.ProvisioningMode) != ProvisioningModeBYOC {
		return sqlc.Trunk{}, apperror.NewConflict("managed trunk endpoints are platform-managed")
	}
	return item, nil
}

func (s *Service) newManagedSIPCredential() (SIPCredential, string, error) {
	usernameEntropy := make([]byte, 12)
	if _, err := rand.Read(usernameEntropy); err != nil {
		return SIPCredential{}, "", err
	}
	passwordEntropy := make([]byte, 32)
	if _, err := rand.Read(passwordEntropy); err != nil {
		return SIPCredential{}, "", err
	}
	username := "lm_sip_" + base64.RawURLEncoding.EncodeToString(usernameEntropy)
	password := "lm_sip_" + base64.RawURLEncoding.EncodeToString(passwordEntropy)
	credential := SIPCredential{Host: ManagedSIPHost, Port: ManagedSIPPort, Transport: ManagedSIPTransport, Realm: ManagedSIPRealm, Username: username, Password: password}
	return credential, hasher.ComputeHA1MD5(username, credential.Realm, password), nil
}

func (s *Service) rotateCredential(ctx context.Context, trunk sqlc.Trunk, credential SIPCredential, ha1 string) error {
	if s.db == nil || s.outbox == nil {
		return apperror.NewServiceUnavailable("managed SIP credential store is unavailable", nil)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return apperror.NewInternal("begin managed SIP credential rotation", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repo := s.repo.WithTx(tx)
	if _, err := repo.RotateCredential(ctx, *trunk.OrganizationID, trunk.ID, credential.Username, credential.Realm, ha1); err != nil {
		return writeError(err, "managed SIP credential", "managed SIP credential not found")
	}
	occurredAt := time.Now().UTC()
	if _, err := s.outbox.WithTx(tx).Insert(ctx, outbox.Event{Subject: string(EventTrunkCredentialRotated), AggregateType: "trunk", AggregateID: trunk.ID, Payload: Event{EventType: EventTrunkCredentialRotated, OrganizationID: *trunk.OrganizationID, TrunkID: trunk.ID, Resource: response(trunk), OccurredAt: occurredAt}, Headers: eventHeaders(EventTrunkCredentialRotated, *trunk.OrganizationID)}); err != nil {
		return apperror.NewInternal("insert managed SIP credential rotation event", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return apperror.NewInternal("commit managed SIP credential rotation", err)
	}
	return nil
}

func (s *Service) mutateTrunk(ctx context.Context, eventType EventType, mutation func(*Repository) (sqlc.Trunk, error)) (sqlc.Trunk, error) {
	if s.db == nil || s.outbox == nil {
		return sqlc.Trunk{}, fmt.Errorf("trunk database is required for domain events")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return sqlc.Trunk{}, fmt.Errorf("begin trunk transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := mutation(s.repo.WithTx(tx))
	if err != nil {
		return sqlc.Trunk{}, err
	}
	if item.OrganizationID == nil {
		return sqlc.Trunk{}, fmt.Errorf("tenant trunk missing organization ownership")
	}
	organizationID := *item.OrganizationID
	occurredAt := time.Now().UTC()
	if _, err := s.outbox.WithTx(tx).Insert(ctx, outbox.Event{Subject: string(eventType), AggregateType: "trunk", AggregateID: item.ID, Payload: Event{EventType: eventType, OrganizationID: organizationID, TrunkID: item.ID, Resource: response(item), OccurredAt: occurredAt}, Headers: eventHeaders(eventType, organizationID)}); err != nil {
		return sqlc.Trunk{}, fmt.Errorf("insert trunk outbox event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Trunk{}, fmt.Errorf("commit trunk transaction: %w", err)
	}
	return item, nil
}

func (s *Service) mutateEndpoint(ctx context.Context, eventType EventType, mutation func(*Repository) (sqlc.TrunkEndpoint, error)) (sqlc.TrunkEndpoint, error) {
	if s.db == nil || s.outbox == nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf("trunk database is required for domain events")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf("begin trunk endpoint transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := mutation(s.repo.WithTx(tx))
	if err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	if item.OrganizationID == nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf("tenant trunk endpoint missing organization ownership")
	}
	organizationID := *item.OrganizationID
	occurredAt := time.Now().UTC()
	endpointID := item.ID
	if _, err := s.outbox.WithTx(tx).Insert(ctx, outbox.Event{Subject: string(eventType), AggregateType: "trunk_endpoint", AggregateID: item.ID, Payload: Event{EventType: eventType, OrganizationID: organizationID, TrunkID: item.TrunkID, EndpointID: &endpointID, Resource: endpointResponse(item), OccurredAt: occurredAt}, Headers: eventHeaders(eventType, organizationID)}); err != nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf("insert trunk endpoint outbox event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf("commit trunk endpoint transaction: %w", err)
	}
	return item, nil
}

func eventHeaders(eventType EventType, organizationID uuid.UUID) map[string]string {
	return map[string]string{"event_type": string(eventType), "organization_id": organizationID.String(), "schema_version": "1"}
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

func writeError(err error, resource, notFound string) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.NewConflict(resource + " already exists")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(notFound)
	}
	return apperror.NewInternal("write "+resource, err)
}

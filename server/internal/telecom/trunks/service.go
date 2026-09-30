package trunks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.Trunk, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return sqlc.Trunk{}, err
	}
	if req.CarrierConnectionID == nil {
		return sqlc.Trunk{}, apperror.NewBadRequest("carrier_connection_id is required")
	}
	if err := validateID(*req.CarrierConnectionID, "carrier_connection_id"); err != nil {
		return sqlc.Trunk{}, err
	}
	name, err := normalizeName(req.Name)
	if err != nil {
		return sqlc.Trunk{}, err
	}
	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil { return sqlc.Trunk{}, err }
		req.Direction = &value
	}
	if req.Status != nil {
		value, err := normalizeChoice(*req.Status, statuses, "status")
		if err != nil { return sqlc.Trunk{}, err }
		req.Status = &value
	}
	item, err := s.mutateTrunk(ctx, EventTrunkCreated, func(repo *Repository) (sqlc.Trunk, error) {
		return repo.Create(ctx, sqlc.CreateTrunkParams{
			OrganizationID: &organizationID,
			CarrierConnectionID: *req.CarrierConnectionID,
			Name: name,
			Direction: req.Direction,
			Status: req.Status,
		})
	})
	return item, writeError(err, "trunk", "carrier connection not found")
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
	if _, err := s.Get(ctx, organizationID, trunkID); err != nil {
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
	if _, err := s.Get(ctx, organizationID, trunkID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListEndpoints(ctx, organizationID, trunkID)
	if err != nil {
		return nil, apperror.NewInternal("list trunk endpoints", err)
	}
	return items, nil
}

func (s *Service) GetEndpoint(ctx context.Context, organizationID, trunkID, id uuid.UUID) (sqlc.TrunkEndpoint, error) {
	if _, err := s.Get(ctx, organizationID, trunkID); err != nil {
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

func (s *Service) Get(ctx context.Context, organizationID, trunkID uuid.UUID) (sqlc.Trunk, error) {
	item, err := s.Get(ctx, organizationID, trunkID)
	if err != nil {
		return sqlc.Trunk{}, err
	}
	if ProvisioningMode(item.ProvisioningMode) != ProvisioningModeBYOC {
		return sqlc.Trunk{}, apperror.NewConflict("managed trunk endpoints are platform-managed")
	}
	return item, nil
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

package trunks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/hasher"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo   *Repository
	db     *pgxpool.Pool
	outbox *outbox.Repository
	cipher *encryption.Cipher
}

func NewService(
	repo *Repository,
	db *pgxpool.Pool,
	cipher *encryption.Cipher,
) *Service {
	service := &Service{
		repo:   repo,
		db:     db,
		cipher: cipher,
	}
	if db != nil {
		service.outbox = outbox.NewRepository(sqlc.New(db))
	}
	return service
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (sqlc.Trunk, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return sqlc.Trunk{}, err
	}
	if err := normalizeCreate(&req); err != nil {
		return sqlc.Trunk{}, err
	}
	if s.db == nil || s.cipher == nil {
		return sqlc.Trunk{}, apperror.NewServiceUnavailable(
			"trunk credential store is unavailable",
			nil,
		)
	}

	id := uuid.New()
	outboundMethod := "none"

	var (
		outboundUsername   *string
		outboundRealm      *string
		outboundCiphertext *string
		outboundHA1        *string

		inboundUsername   *string
		inboundRealm      *string
		inboundCiphertext *string
		inboundHA1        *string
	)

	if req.OutboundCredential != nil {
		outboundMethod = "digest"
		value, err := s.cipher.EncryptForScope(
			credentialScope(organizationID, id, "outbound"),
			req.OutboundCredential.Secret,
		)
		if err != nil {
			return sqlc.Trunk{}, apperror.NewInternal(
				"encrypt outbound trunk credential",
				err,
			)
		}
		outboundUsername = &req.OutboundCredential.Username
		outboundRealm = &req.OutboundCredential.Realm
		outboundCiphertext = &value
		ha1 := hasher.ComputeHA1MD5(
			req.OutboundCredential.Username,
			req.OutboundCredential.Realm,
			req.OutboundCredential.Secret,
		)
		outboundHA1 = &ha1
	}

	if req.InboundCredential != nil {
		value, err := s.cipher.EncryptForScope(
			credentialScope(organizationID, id, "inbound"),
			req.InboundCredential.Secret,
		)
		if err != nil {
			return sqlc.Trunk{}, apperror.NewInternal(
				"encrypt inbound trunk credential",
				err,
			)
		}
		inboundUsername = &req.InboundCredential.Username
		inboundRealm = &req.InboundCredential.Realm
		inboundCiphertext = &value
		ha1 := hasher.ComputeHA1MD5(
			req.InboundCredential.Username,
			req.InboundCredential.Realm,
			req.InboundCredential.Secret,
		)
		inboundHA1 = &ha1
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return sqlc.Trunk{}, apperror.NewInternal(
			"begin trunk creation",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	transactionRepo := s.repo.WithTx(tx)
	item, err := transactionRepo.Create(ctx, sqlc.CreateTrunkParams{
		ID:                      id,
		OrganizationID:          organizationID,
		Name:                    req.Name,
		Direction:               req.Direction,
		Status:                  req.Status,
		OutboundAuthMethod:      &outboundMethod,
		AuthUsername:            outboundUsername,
		AuthRealm:               outboundRealm,
		AuthSecretCiphertext:    outboundCiphertext,
		InboundEnabled:          req.InboundEnabled,
		InboundAuthMethod:       req.InboundAuthMethod,
		InboundUsername:         inboundUsername,
		InboundRealm:            inboundRealm,
		InboundSecretCiphertext: inboundCiphertext,
		MaxCps:                  req.MaxCPS,
		MaxConcurrentCalls:      req.MaxConcurrentCalls,
		Codecs:                  req.Codecs,
		SupportsVideo:           req.SupportsVideo,
		SupportsFax:             req.SupportsFax,
	})
	if err != nil {
		return sqlc.Trunk{}, writeError(
			err,
			"trunk could not be created",
		)
	}

	if outboundHA1 != nil {
		if err := transactionRepo.InsertDigest(
			ctx,
			organizationID,
			id,
			"outbound",
			*outboundUsername,
			*outboundRealm,
			*outboundHA1,
		); err != nil {
			return sqlc.Trunk{}, writeError(
				err,
				"outbound trunk credentials could not be created",
			)
		}
	}
	if inboundHA1 != nil {
		if err := transactionRepo.InsertDigest(
			ctx,
			organizationID,
			id,
			"inbound",
			*inboundUsername,
			*inboundRealm,
			*inboundHA1,
		); err != nil {
			return sqlc.Trunk{}, writeError(
				err,
				"inbound trunk credentials could not be created",
			)
		}
	}

	if err := s.insertEvent(
		ctx,
		tx,
		EventTrunkCreated,
		item,
		nil,
		response(item),
	); err != nil {
		return sqlc.Trunk{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Trunk{}, apperror.NewInternal(
			"commit trunk creation",
			err,
		)
	}
	return item, nil
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Trunk, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return nil, err
	}
	items, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list trunks", err)
	}
	return items, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Trunk, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return sqlc.Trunk{}, err
	}
	if err := validateID(id, "trunk_id"); err != nil {
		return sqlc.Trunk{}, err
	}
	item, err := s.repo.Get(ctx, organizationID, id)
	return item, readError(err, "trunk not found")
}

func (s *Service) Validate(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (ValidationResponse, error) {
	trunk, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return ValidationResponse{}, err
	}

	result := ValidationResponse{
		Errors:   []string{},
		Warnings: []string{},
	}

	if trunk.OutboundAuthMethod == "digest" &&
		(trunk.AuthSecretCiphertext == nil || trunk.AuthRealm == nil) {
		result.Errors = append(
			result.Errors,
			"outbound digest credentials and realm are missing",
		)
	}

	if trunk.InboundEnabled {
		switch trunk.InboundAuthMethod {
		case "digest":
			if trunk.InboundSecretCiphertext == nil ||
				trunk.InboundRealm == nil {
				result.Errors = append(
					result.Errors,
					"inbound digest credentials and realm are missing",
				)
			}
		case "ip":
			sourceIPs, err := s.ListSourceIPs(
				ctx,
				organizationID,
				id,
			)
			if err != nil {
				return ValidationResponse{}, err
			}
			if len(sourceIPs) == 0 {
				result.Errors = append(
					result.Errors,
					"inbound IP authentication requires at least one source CIDR",
				)
			}
		case "none":
			result.Warnings = append(
				result.Warnings,
				"inbound traffic is enabled without authentication",
			)
		default:
			result.Errors = append(
				result.Errors,
				"inbound authentication method is invalid",
			)
		}
	}

	if trunk.Status != "active" {
		result.Warnings = append(result.Warnings, "trunk is disabled")
	}

	endpoints, err := s.repo.ListEndpoints(ctx, organizationID, id)
	if err != nil {
		return ValidationResponse{}, apperror.NewInternal(
			"list trunk endpoints",
			err,
		)
	}

	if trunk.Direction == "outbound" ||
		trunk.Direction == "bidirectional" {
		enabled := 0
		routable := 0
		for _, endpoint := range endpoints {
			if !endpoint.Enabled ||
				(endpoint.Direction != "outbound" &&
					endpoint.Direction != "bidirectional") {
				continue
			}
			enabled++
			if endpoint.HealthStatus != "unhealthy" {
				routable++
			}
		}
		if enabled == 0 {
			result.Errors = append(
				result.Errors,
				"trunk has no enabled outbound endpoint",
			)
		} else if routable == 0 {
			result.Errors = append(
				result.Errors,
				"trunk has no healthy outbound endpoint",
			)
		}
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

func (s *Service) Update(
	ctx context.Context,
	organizationID, id uuid.UUID,
	req UpdateRequest,
) (sqlc.Trunk, error) {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return sqlc.Trunk{}, err
	}
	if err := normalizeUpdate(&req); err != nil {
		return sqlc.Trunk{}, err
	}

	item, err := s.mutateTrunk(
		ctx,
		EventTrunkUpdated,
		func(repo *Repository) (sqlc.Trunk, error) {
			var codecs []string
			if req.Codecs != nil {
				codecs = *req.Codecs
			}
			return repo.Update(ctx, sqlc.UpdateTrunkParams{
				Name:               req.Name,
				Direction:          req.Direction,
				Status:             req.Status,
				InboundEnabled:     req.InboundEnabled,
				MaxCps:             req.MaxCPS,
				MaxConcurrentCalls: req.MaxConcurrentCalls,
				Codecs:             codecs,
				SupportsVideo:      req.SupportsVideo,
				SupportsFax:        req.SupportsFax,
				ID:                 id,
				OrganizationID:     organizationID,
			})
		},
	)
	return item, writeError(err, "trunk not found")
}

func (s *Service) Delete(
	ctx context.Context,
	organizationID, id uuid.UUID,
) error {
	item, err := s.Get(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if item.Status == "disabled" {
		return nil
	}

	_, err = s.mutateTrunk(
		ctx,
		EventTrunkDisabled,
		func(repo *Repository) (sqlc.Trunk, error) {
			return repo.Disable(ctx, organizationID, id)
		},
	)
	return writeError(err, "trunk not found")
}

func (s *Service) SetOutboundAuth(
	ctx context.Context,
	organizationID, id uuid.UUID,
	req AuthRequest,
) error {
	if err := normalizeAuth(&req, false); err != nil {
		return err
	}
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	if req.Method == "none" {
		return s.repo.ClearOutbound(ctx, organizationID, id)
	}
	if s.cipher == nil {
		return apperror.NewServiceUnavailable(
			"trunk credential store is unavailable",
			nil,
		)
	}

	encrypted, err := s.cipher.EncryptForScope(
		credentialScope(organizationID, id, "outbound"),
		*req.Secret,
	)
	if err != nil {
		return apperror.NewInternal(
			"encrypt outbound trunk credential",
			err,
		)
	}

	return s.repo.SetOutboundDigest(
		ctx,
		organizationID,
		id,
		*req.Username,
		*req.Realm,
		encrypted,
		hasher.ComputeHA1MD5(
			*req.Username,
			*req.Realm,
			*req.Secret,
		),
	)
}

func (s *Service) ClearOutboundAuth(
	ctx context.Context,
	organizationID, id uuid.UUID,
) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	return s.repo.ClearOutbound(ctx, organizationID, id)
}

func (s *Service) SetInboundAuth(
	ctx context.Context,
	organizationID, id uuid.UUID,
	req AuthRequest,
) error {
	if err := normalizeAuth(&req, true); err != nil {
		return err
	}
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}

	switch req.Method {
	case "ip":
		return s.repo.SetInboundIP(ctx, organizationID, id)
	case "none":
		return s.repo.SetInboundNone(ctx, organizationID, id)
	}

	if s.cipher == nil {
		return apperror.NewServiceUnavailable(
			"trunk credential store is unavailable",
			nil,
		)
	}

	encrypted, err := s.cipher.EncryptForScope(
		credentialScope(organizationID, id, "inbound"),
		*req.Secret,
	)
	if err != nil {
		return apperror.NewInternal(
			"encrypt inbound trunk credential",
			err,
		)
	}

	return s.repo.SetInboundDigest(
		ctx,
		organizationID,
		id,
		*req.Username,
		*req.Realm,
		encrypted,
		hasher.ComputeHA1MD5(
			*req.Username,
			*req.Realm,
			*req.Secret,
		),
	)
}

func (s *Service) ClearInboundAuth(
	ctx context.Context,
	organizationID, id uuid.UUID,
) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	return s.repo.SetInboundNone(ctx, organizationID, id)
}

func (s *Service) CreateSourceIP(
	ctx context.Context,
	organizationID, id uuid.UUID,
	value string,
) (SourceIPResponse, error) {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return SourceIPResponse{}, err
	}
	cidr, err := parseCIDR(value)
	if err != nil {
		return SourceIPResponse{}, err
	}

	row, err := s.repo.CreateSourceIP(
		ctx,
		organizationID,
		id,
		cidr,
	)
	if err != nil {
		return SourceIPResponse{}, writeError(
			err,
			"trunk source IP could not be created",
		)
	}
	return sourceIPResponse(row), nil
}

func (s *Service) ListSourceIPs(
	ctx context.Context,
	organizationID, id uuid.UUID,
) ([]SourceIPResponse, error) {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListSourceIPs(ctx, organizationID, id)
	if err != nil {
		return nil, apperror.NewInternal(
			"list trunk source IPs",
			err,
		)
	}

	result := make([]SourceIPResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, sourceIPResponse(row))
	}
	return result, nil
}

func (s *Service) DeleteSourceIP(
	ctx context.Context,
	organizationID, id, sourceIPID uuid.UUID,
) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	if sourceIPID == uuid.Nil {
		return apperror.NewBadRequest("source_ip_id is required")
	}
	return s.repo.DeleteSourceIP(
		ctx,
		organizationID,
		id,
		sourceIPID,
	)
}

func (s *Service) CreateEndpoint(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
	req EndpointCreateRequest,
) (sqlc.TrunkEndpoint, error) {
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
		value, err := normalizeChoice(
			*req.Transport,
			transports,
			"transport",
		)
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Transport = &value
	}
	if req.Direction != nil {
		value, err := normalizeChoice(
			*req.Direction,
			directions,
			"direction",
		)
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

	item, err := s.mutateEndpoint(
		ctx,
		EventTrunkEndpointCreated,
		func(repo *Repository) (sqlc.TrunkEndpoint, error) {
			return repo.CreateEndpoint(
				ctx,
				sqlc.CreateTrunkEndpointParams{
					OrganizationID: organizationID,
					TrunkID:        trunkID,
					Host:           host,
					Port:           req.Port,
					Transport:      req.Transport,
					Direction:      req.Direction,
					Priority:       req.Priority,
					Weight:         req.Weight,
					Enabled:        req.Enabled,
				},
			)
		},
	)
	return item, writeError(err, "trunk endpoint not found")
}

func (s *Service) ListEndpoints(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
) ([]sqlc.TrunkEndpoint, error) {
	if _, err := s.Get(ctx, organizationID, trunkID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListEndpoints(ctx, organizationID, trunkID)
	if err != nil {
		return nil, apperror.NewInternal("list trunk endpoints", err)
	}
	return items, nil
}

func (s *Service) GetEndpoint(
	ctx context.Context,
	organizationID, trunkID, id uuid.UUID,
) (sqlc.TrunkEndpoint, error) {
	if _, err := s.Get(ctx, organizationID, trunkID); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	if err := validateID(id, "endpoint_id"); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}
	item, err := s.repo.GetEndpoint(
		ctx,
		organizationID,
		trunkID,
		id,
	)
	return item, readError(err, "trunk endpoint not found")
}

func (s *Service) UpdateEndpoint(
	ctx context.Context,
	organizationID, trunkID, id uuid.UUID,
	req EndpointUpdateRequest,
) (sqlc.TrunkEndpoint, error) {
	if _, err := s.GetEndpoint(
		ctx,
		organizationID,
		trunkID,
		id,
	); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}

	if req.Host == nil &&
		req.Port == nil &&
		req.Transport == nil &&
		req.Direction == nil &&
		req.Priority == nil &&
		req.Weight == nil &&
		req.Enabled == nil {
		return sqlc.TrunkEndpoint{}, apperror.NewBadRequest(
			"at least one field is required",
		)
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
		value, err := normalizeChoice(
			*req.Transport,
			transports,
			"transport",
		)
		if err != nil {
			return sqlc.TrunkEndpoint{}, err
		}
		req.Transport = &value
	}
	if req.Direction != nil {
		value, err := normalizeChoice(
			*req.Direction,
			directions,
			"direction",
		)
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

	item, err := s.mutateEndpoint(
		ctx,
		EventTrunkEndpointUpdated,
		func(repo *Repository) (sqlc.TrunkEndpoint, error) {
			return repo.UpdateEndpoint(
				ctx,
				sqlc.UpdateTrunkEndpointParams{
					Host:           req.Host,
					Port:           req.Port,
					Transport:      req.Transport,
					Direction:      req.Direction,
					Priority:       req.Priority,
					Weight:         req.Weight,
					Enabled:        req.Enabled,
					ID:             id,
					TrunkID:        trunkID,
					OrganizationID: organizationID,
				},
			)
		},
	)
	return item, writeError(err, "trunk endpoint not found")
}

func (s *Service) DeleteEndpoint(
	ctx context.Context,
	organizationID, trunkID, id uuid.UUID,
) error {
	if _, err := s.GetEndpoint(
		ctx,
		organizationID,
		trunkID,
		id,
	); err != nil {
		return err
	}

	_, err := s.mutateEndpoint(
		ctx,
		EventTrunkEndpointDeleted,
		func(repo *Repository) (sqlc.TrunkEndpoint, error) {
			return repo.DeleteEndpoint(
				ctx,
				organizationID,
				trunkID,
				id,
			)
		},
	)
	return writeError(err, "trunk endpoint not found")
}

func (s *Service) mutateTrunk(
	ctx context.Context,
	eventType EventType,
	mutation func(*Repository) (sqlc.Trunk, error),
) (sqlc.Trunk, error) {
	if s.db == nil || s.outbox == nil {
		return sqlc.Trunk{}, fmt.Errorf(
			"trunk database is required for domain events",
		)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return sqlc.Trunk{}, fmt.Errorf(
			"begin trunk transaction: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	item, err := mutation(s.repo.WithTx(tx))
	if err != nil {
		return sqlc.Trunk{}, err
	}

	if err := s.insertEvent(
		ctx,
		tx,
		eventType,
		item,
		nil,
		response(item),
	); err != nil {
		return sqlc.Trunk{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Trunk{}, fmt.Errorf(
			"commit trunk transaction: %w",
			err,
		)
	}
	return item, nil
}

func (s *Service) mutateEndpoint(
	ctx context.Context,
	eventType EventType,
	mutation func(*Repository) (sqlc.TrunkEndpoint, error),
) (sqlc.TrunkEndpoint, error) {
	if s.db == nil || s.outbox == nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf(
			"trunk database is required for domain events",
		)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf(
			"begin trunk endpoint transaction: %w",
			err,
		)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	item, err := mutation(s.repo.WithTx(tx))
	if err != nil {
		return sqlc.TrunkEndpoint{}, err
	}

	trunk, err := s.repo.WithTx(tx).Get(
		ctx,
		item.OrganizationID,
		item.TrunkID,
	)
	if err != nil {
		return sqlc.TrunkEndpoint{}, err
	}

	endpointID := item.ID
	if err := s.insertEvent(
		ctx,
		tx,
		eventType,
		trunk,
		&endpointID,
		endpointResponse(item),
	); err != nil {
		return sqlc.TrunkEndpoint{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.TrunkEndpoint{}, fmt.Errorf(
			"commit trunk endpoint transaction: %w",
			err,
		)
	}
	return item, nil
}

func (s *Service) insertEvent(
	ctx context.Context,
	tx pgx.Tx,
	eventType EventType,
	trunk sqlc.Trunk,
	endpointID *uuid.UUID,
	resource any,
) error {
	if _, err := s.outbox.WithTx(tx).Insert(ctx, outbox.Event{
		Subject:       string(eventType),
		AggregateType: "trunk",
		AggregateID:   trunk.ID,
		Payload: Event{
			EventType:      eventType,
			OrganizationID: trunk.OrganizationID,
			TrunkID:        trunk.ID,
			EndpointID:     endpointID,
			Resource:       resource,
			OccurredAt:     time.Now().UTC(),
		},
		Headers: map[string]string{
			"event_type":      string(eventType),
			"organization_id": trunk.OrganizationID.String(),
			"schema_version":  "1",
		},
	}); err != nil {
		return fmt.Errorf("insert trunk outbox event: %w", err)
	}
	return nil
}

func credentialScope(
	organizationID, trunkID uuid.UUID,
	direction string,
) string {
	return fmt.Sprintf(
		"organization:%s:trunk:%s:%s",
		organizationID,
		trunkID,
		direction,
	)
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
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperror.NewConflict("trunk resource already exists")
		case "23503", "23514", "23502":
			return apperror.NewBadRequest(message)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}

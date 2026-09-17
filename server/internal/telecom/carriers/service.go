package carriers

import (
	"context"
	"errors"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo   *Repository
	db     *pgxpool.Pool
	cipher *encryption.Cipher
}

func NewService(repo *Repository, db *pgxpool.Pool, cipher *encryption.Cipher) *Service {
	return &Service{repo: repo, db: db, cipher: cipher}
}

func (s *Service) Create(ctx context.Context, org uuid.UUID, req CreateRequest) (Response, error) {
	if org == uuid.Nil {
		return Response{}, apperror.NewBadRequest("organization context required")
	}
	if err := normalizeCreate(&req); err != nil {
		return Response{}, apperror.NewBadRequest(err.Error())
	}
	if s.db == nil || s.cipher == nil {
		return Response{}, apperror.NewServiceUnavailable("carrier credential store is unavailable", nil)
	}
	id := uuid.New()
	outMethod := "none"
	var outUser, outCipher *string
	if req.OutboundCredential != nil {
		outMethod = "digest"
		value, err := s.cipher.EncryptForScope(credentialScope(org, id, "outbound"), req.OutboundCredential.Secret)
		if err != nil {
			return Response{}, apperror.NewInternal("encrypt outbound carrier credential", err)
		}
		outUser = &req.OutboundCredential.Username
		outCipher = &value
	}
	var inUser, inCipher *string
	if req.InboundCredential != nil {
		value, err := s.cipher.EncryptForScope(credentialScope(org, id, "inbound"), req.InboundCredential.Secret)
		if err != nil {
			return Response{}, apperror.NewInternal("encrypt inbound carrier credential", err)
		}
		inUser = &req.InboundCredential.Username
		inCipher = &value
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Response{}, apperror.NewInternal("begin carrier connection creation", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := s.repo.WithTx(tx).Create(ctx, sqlc.CreateCarrierConnectionParams{ID: id, OrganizationID: &org, ProviderID: req.ProviderID, Name: req.Name, Status: req.Status, OutboundAuthMethod: &outMethod, AuthUsername: outUser, AuthSecretCiphertext: outCipher, InboundEnabled: req.InboundEnabled, InboundAuthMethod: req.InboundAuthMethod, InboundUsername: inUser, InboundSecretCiphertext: inCipher, MaxCps: req.MaxCPS, MaxConcurrentCalls: req.MaxConcurrentCalls, MaxDailyMinutes: req.MaxDailyMinutes, Codecs: req.Codecs, SupportsVideo: req.SupportsVideo, SupportsFax: req.SupportsFax})
	if err != nil {
		return Response{}, writeError(err, "carrier connection could not be created")
	}
	if err = tx.Commit(ctx); err != nil {
		return Response{}, apperror.NewInternal("commit carrier connection creation", err)
	}
	return responseFromRow(row), nil
}

func (s *Service) Get(ctx context.Context, org, id uuid.UUID) (Response, error) {
	if err := validIDs(org, id); err != nil {
		return Response{}, err
	}
	row, err := s.repo.Get(ctx, org, id)
	if err != nil {
		return Response{}, readError(err)
	}
	return responseFromGet(row), nil
}
func (s *Service) List(ctx context.Context, org uuid.UUID) ([]Response, error) {
	if org == uuid.Nil {
		return nil, apperror.NewBadRequest("organization context required")
	}
	rows, err := s.repo.List(ctx, org)
	if err != nil {
		return nil, apperror.NewInternal("list carrier connections", err)
	}
	result := make([]Response, 0, len(rows))
	for _, row := range rows {
		result = append(result, responseFromList(row))
	}
	return result, nil
}
func (s *Service) Update(ctx context.Context, org, id uuid.UUID, req UpdateRequest) (Response, error) {
	if err := validIDs(org, id); err != nil {
		return Response{}, err
	}
	if err := normalizeUpdate(&req); err != nil {
		return Response{}, apperror.NewBadRequest(err.Error())
	}
	row, err := s.repo.Update(ctx, org, id, req)
	if err != nil {
		return Response{}, writeError(err, "carrier connection not found")
	}
	return responseFromRow(row), nil
}
func (s *Service) SetOutboundAuth(ctx context.Context, org, id uuid.UUID, req AuthRequest) error {
	if err := validIDs(org, id); err != nil {
		return err
	}
	if err := normalizeAuth(&req, false); err != nil {
		return apperror.NewBadRequest(err.Error())
	}
	if s.cipher == nil {
		return apperror.NewServiceUnavailable("carrier credential store is unavailable", nil)
	}
	if _, err := s.repo.Get(ctx, org, id); err != nil {
		return readError(err)
	}
	if req.Method == "none" {
		return s.repo.ClearOutbound(ctx, org, id)
	}
	encrypted, err := s.cipher.EncryptForScope(credentialScope(org, id, "outbound"), *req.Secret)
	if err != nil {
		return apperror.NewInternal("encrypt outbound carrier credential", err)
	}
	return s.repo.SetOutboundDigest(ctx, org, id, *req.Username, encrypted)
}
func (s *Service) SetInboundAuth(ctx context.Context, org, id uuid.UUID, req AuthRequest) error {
	if err := validIDs(org, id); err != nil {
		return err
	}
	if err := normalizeAuth(&req, true); err != nil {
		return apperror.NewBadRequest(err.Error())
	}
	if s.cipher == nil {
		return apperror.NewServiceUnavailable("carrier credential store is unavailable", nil)
	}
	if _, err := s.repo.Get(ctx, org, id); err != nil {
		return readError(err)
	}
	switch req.Method {
	case "ip":
		return s.repo.SetInboundIP(ctx, org, id)
	case "none":
		return s.repo.SetInboundNone(ctx, org, id)
	}
	encrypted, err := s.cipher.EncryptForScope(credentialScope(org, id, "inbound"), *req.Secret)
	if err != nil {
		return apperror.NewInternal("encrypt inbound carrier credential", err)
	}
	return s.repo.SetInboundDigest(ctx, org, id, *req.Username, encrypted)
}
func (s *Service) AddSourceIP(ctx context.Context, org, id uuid.UUID, value string) (SourceIPResponse, error) {
	if err := validIDs(org, id); err != nil {
		return SourceIPResponse{}, err
	}
	cidr, err := parseCIDR(value)
	if err != nil {
		return SourceIPResponse{}, apperror.NewBadRequest(err.Error())
	}
	row, err := s.repo.AddSourceIP(ctx, org, id, cidr)
	if err != nil {
		return SourceIPResponse{}, writeError(err, "carrier connection not found")
	}
	return sourceIPResponse(row), nil
}
func (s *Service) ListSourceIPs(ctx context.Context, org, id uuid.UUID) ([]SourceIPResponse, error) {
	if err := validIDs(org, id); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListSourceIPs(ctx, org, id)
	if err != nil {
		return nil, apperror.NewInternal("list carrier source IPs", err)
	}
	result := make([]SourceIPResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, sourceIPResponse(row))
	}
	return result, nil
}
func (s *Service) DeleteSourceIP(ctx context.Context, org, id, sourceID uuid.UUID) error {
	if err := validIDs(org, id); err != nil {
		return err
	}
	if sourceID == uuid.Nil {
		return apperror.NewBadRequest("source_ip_id is required")
	}
	return s.repo.DeleteSourceIP(ctx, org, id, sourceID)
}

func credentialScope(org, id uuid.UUID, direction string) string {
	return fmt.Sprintf("organization:%s:carrier-connection:%s:%s", org, id, direction)
}
func validIDs(org, id uuid.UUID) error {
	if org == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("carrier_connection_id is required")
	}
	return nil
}
func readError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("carrier connection not found")
	}
	return apperror.NewInternal("get carrier connection", err)
}
func writeError(err error, message string) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		switch pgerr.Code {
		case "23505":
			return apperror.NewConflict("carrier connection already exists")
		case "23503", "23514":
			return apperror.NewBadRequest(message)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}

func responseFromRow(r sqlc.CarrierConnection) Response {
	org := uuid.Nil
	if r.OrganizationID != nil {
		org = *r.OrganizationID
	}
	return Response{ID: r.ID, OrganizationID: org, ProviderID: r.ProviderID, Name: r.Name, Status: r.Status, OutboundAuthMethod: r.OutboundAuthMethod, OutboundUsername: r.AuthUsername, HasOutboundCredentials: r.AuthSecretCiphertext != nil, InboundEnabled: r.InboundEnabled, InboundAuthMethod: r.InboundAuthMethod, InboundUsername: r.InboundUsername, HasInboundCredentials: r.InboundSecretCiphertext != nil, MaxCPS: r.MaxCps, MaxConcurrentCalls: r.MaxConcurrentCalls, MaxDailyMinutes: r.MaxDailyMinutes, Codecs: r.Codecs, SupportsVideo: r.SupportsVideo, SupportsFax: r.SupportsFax, CreatedAt: pgconv.TimestamptzToTime(r.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(r.UpdatedAt)}
}
func responseFromGet(r sqlc.GetCarrierConnectionByIDRow) Response {
	org := uuid.Nil
	if r.OrganizationID != nil {
		org = *r.OrganizationID
	}
	return Response{ID: r.ID, OrganizationID: org, ProviderID: r.ProviderID, Name: r.Name, Status: r.Status, OutboundAuthMethod: r.OutboundAuthMethod, OutboundUsername: r.AuthUsername, HasOutboundCredentials: boolValue(r.HasOutboundCredentials), InboundEnabled: r.InboundEnabled, InboundAuthMethod: r.InboundAuthMethod, InboundUsername: r.InboundUsername, HasInboundCredentials: boolValue(r.HasInboundCredentials), MaxCPS: r.MaxCps, MaxConcurrentCalls: r.MaxConcurrentCalls, MaxDailyMinutes: r.MaxDailyMinutes, Codecs: r.Codecs, SupportsVideo: r.SupportsVideo, SupportsFax: r.SupportsFax, CreatedAt: pgconv.TimestamptzToTime(r.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(r.UpdatedAt)}
}
func responseFromList(r sqlc.ListCarrierConnectionsByOrganizationIDRow) Response {
	return Response{ID: r.ID, OrganizationID: *r.OrganizationID, ProviderID: r.ProviderID, Name: r.Name, Status: r.Status, OutboundAuthMethod: r.OutboundAuthMethod, OutboundUsername: r.AuthUsername, HasOutboundCredentials: boolValue(r.HasOutboundCredentials), InboundEnabled: r.InboundEnabled, InboundAuthMethod: r.InboundAuthMethod, InboundUsername: r.InboundUsername, HasInboundCredentials: boolValue(r.HasInboundCredentials), MaxCPS: r.MaxCps, MaxConcurrentCalls: r.MaxConcurrentCalls, MaxDailyMinutes: r.MaxDailyMinutes, Codecs: r.Codecs, SupportsVideo: r.SupportsVideo, SupportsFax: r.SupportsFax, CreatedAt: pgconv.TimestamptzToTime(r.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(r.UpdatedAt)}
}
func sourceIPResponse(r sqlc.CarrierConnectionSourceIp) SourceIPResponse {
	return SourceIPResponse{ID: r.ID, CarrierConnectionID: r.CarrierConnectionID, CIDR: r.Cidr, CreatedAt: pgconv.TimestamptzToTime(r.CreatedAt)}
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

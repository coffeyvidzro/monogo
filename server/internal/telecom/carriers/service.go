package carriers

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/modules/audit"
	"github.com/leamout/leamout/internal/telecom/routing"
	"github.com/leamout/leamout/pkg/apperror"
)

// credentialSealer is intentionally encryption-only. The customer-facing
// carrier service can replace or revoke a credential, but it must never recover
// or export its plaintext. SIP Digest execution uses the separately persisted
// realm-bound HA1 value. A future protocol that genuinely requires plaintext
// must decrypt it in a narrower execution component, not by broadening this
// API-facing service's authority.
type credentialSealer interface {
	EncryptForScope(string, string) (string, error)
}

type Service struct {
	repo   *Repository
	cipher credentialSealer
	prober routing.EndpointProber
}

func NewService(repo *Repository, cipher credentialSealer) *Service {
	return &Service{repo: repo, cipher: cipher, prober: routing.NewSIPOptionsProber()}
}

// Validate checks that a carrier connection has an eligible outbound topology
// and performs bounded SIP OPTIONS probes without changing circuit-breaker
// state. It is an explicit operator workflow, not a substitute for the worker's
// continuous endpoint health checks.
func (s *Service) Validate(ctx context.Context, organizationID, connectionID uuid.UUID) (ValidationResponse, error) {
	connection, err := s.Get(ctx, organizationID, connectionID)
	if err != nil {
		return ValidationResponse{}, err
	}
	result := ValidationResponse{CarrierConnectionID: connectionID, Issues: []ValidationIssue{}, Endpoints: []EndpointValidation{}}
	probeRootCtx, cancelProbes := context.WithTimeout(ctx, 10*time.Second)
	defer cancelProbes()
	if connection.Status != "active" {
		result.Issues = append(result.Issues, ValidationIssue{Severity: "error", Code: "connection_inactive", Message: "carrier connection must be active"})
	}
	if connection.OutboundAuthMethod == "digest" && !connection.HasOutboundCredentials {
		result.Issues = append(result.Issues, ValidationIssue{Severity: "error", Code: "outbound_credentials_missing", Message: "digest outbound authentication requires active credentials"})
	}

	trunks, err := s.repo.ListConnectionTrunks(ctx, organizationID, connectionID)
	if err != nil {
		return ValidationResponse{}, apperror.NewInternal("list carrier connection trunks for validation", err)
	}
	eligibleTrunks := 0
	for _, trunk := range trunks {
		if trunk.Status != "active" || (trunk.Direction != "outbound" && trunk.Direction != "bidirectional") {
			continue
		}
		eligibleTrunks++
		endpoints, err := s.repo.ListTrunkEndpoints(ctx, organizationID, trunk.ID)
		if err != nil {
			return ValidationResponse{}, apperror.NewInternal("list trunk endpoints for validation", err)
		}
		eligibleEndpoints := 0
		for _, endpoint := range endpoints {
			if !endpoint.Enabled || (endpoint.Direction != "outbound" && endpoint.Direction != "bidirectional") {
				continue
			}
			eligibleEndpoints++
			check := EndpointValidation{TrunkID: trunk.ID, EndpointID: endpoint.ID, Host: endpoint.Host, Port: endpoint.Port, Transport: endpoint.Transport}
			probeCtx, cancel := context.WithTimeout(probeRootCtx, 2*time.Second)
			probe, probeErr := s.prober.Probe(probeCtx, endpoint)
			cancel()
			check.LatencyMS = probe.Latency.Milliseconds()
			if probeErr != nil {
				check.Error = truncateValidationError(probeErr.Error())
				endpointID, trunkID := endpoint.ID, trunk.ID
				result.Issues = append(result.Issues, ValidationIssue{Severity: "error", Code: "endpoint_unreachable", Message: check.Error, TrunkID: &trunkID, EndpointID: &endpointID})
			} else {
				check.Reachable = true
				check.ResponseCode = &probe.ResponseCode
			}
			result.Endpoints = append(result.Endpoints, check)
		}
		if eligibleEndpoints == 0 {
			trunkID := trunk.ID
			result.Issues = append(result.Issues, ValidationIssue{Severity: "error", Code: "no_eligible_endpoints", Message: "active outbound trunk has no enabled outbound endpoint", TrunkID: &trunkID})
		}
	}
	if eligibleTrunks == 0 {
		result.Issues = append(result.Issues, ValidationIssue{Severity: "error", Code: "no_eligible_trunks", Message: "carrier connection has no active outbound trunk"})
	}
	result.Valid = len(result.Issues) == 0
	return result, nil
}

func truncateValidationError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

func (s *Service) SetOutboundAuth(ctx context.Context, org, id uuid.UUID, req DigestAuthRequest) (Response, error) {
	current, err := s.Get(ctx, org, id)
	if err != nil {
		return Response{}, err
	}
	if strings.ToLower(strings.TrimSpace(req.Method)) != "digest" {
		return Response{}, apperror.NewBadRequest("outbound auth method must be digest")
	}
	username, err := requireAuthValue(req.Username, "username")
	if err != nil {
		return Response{}, err
	}
	secret, err := requireAuthValue(req.Secret, "secret")
	if err != nil {
		return Response{}, err
	}
	ciphertext, err := s.cipher.EncryptForScope(credentialEncryptionScope(org, id, "outbound"), secret)
	if err != nil {
		return Response{}, apperror.NewInternal("encrypt outbound carrier credential", err)
	}
	realm, err := requireAuthValue(req.Realm, "realm")
	if err != nil {
		return Response{}, err
	}
	event, err := credentialAuditEvent(ctx, org, id, "outbound", "digest", current.HasOutboundCredentials)
	if err != nil {
		return Response{}, apperror.NewInternal("attribute outbound credential audit event", err)
	}
	if err := s.repo.SetDigestAuth(ctx, org, id, "outbound", username, realm, ciphertext, digestHA1(username, realm, secret), event); err != nil {
		return Response{}, digestWriteError(err, "set outbound carrier auth")
	}
	return s.Get(ctx, org, id)
}

func (s *Service) ClearOutboundAuth(ctx context.Context, org, id uuid.UUID) error {
	if _, err := s.Get(ctx, org, id); err != nil {
		return err
	}
	event, err := credentialDeletionAuditEvent(ctx, org, id, "outbound")
	if err != nil {
		return apperror.NewInternal("attribute outbound credential deletion", err)
	}
	return writeAuthError(s.repo.ClearAuth(ctx, org, id, "outbound", event), "clear outbound carrier auth")
}

func (s *Service) SetInboundAuth(ctx context.Context, org, id uuid.UUID, req InboundAuthRequest) (Response, error) {
	current, err := s.Get(ctx, org, id)
	if err != nil {
		return Response{}, err
	}
	switch strings.ToLower(strings.TrimSpace(req.Method)) {
	case "ip":
		event, err := credentialAuditEvent(ctx, org, id, "inbound", "ip", current.InboundAuthMethod != "none")
		if err != nil {
			return Response{}, apperror.NewInternal("attribute inbound authentication audit event", err)
		}
		if err := s.repo.SetInboundIPAuth(ctx, org, id, event); err != nil {
			return Response{}, apperror.NewInternal("set inbound carrier IP auth", err)
		}
	case "digest":
		username, err := requireAuthValue(req.Username, "username")
		if err != nil {
			return Response{}, err
		}
		secret, err := requireAuthValue(req.Secret, "secret")
		if err != nil {
			return Response{}, err
		}
		ciphertext, err := s.cipher.EncryptForScope(credentialEncryptionScope(org, id, "inbound"), secret)
		if err != nil {
			return Response{}, apperror.NewInternal("encrypt inbound carrier credential", err)
		}
		realm, err := requireAuthValue(req.Realm, "realm")
		if err != nil {
			return Response{}, err
		}
		event, err := credentialAuditEvent(ctx, org, id, "inbound", "digest", current.InboundAuthMethod != "none")
		if err != nil {
			return Response{}, apperror.NewInternal("attribute inbound credential audit event", err)
		}
		if err := s.repo.SetDigestAuth(ctx, org, id, "inbound", username, realm, ciphertext, digestHA1(username, realm, secret), event); err != nil {
			return Response{}, digestWriteError(err, "set inbound carrier auth")
		}
	default:
		return Response{}, apperror.NewBadRequest("inbound auth method must be digest or ip")
	}
	return s.Get(ctx, org, id)
}

func credentialEncryptionScope(organizationID, connectionID uuid.UUID, direction string) string {
	return "organization/" + organizationID.String() + "/carrier/" + connectionID.String() + "/" + direction
}

func (s *Service) ClearInboundAuth(ctx context.Context, org, id uuid.UUID) error {
	if _, err := s.Get(ctx, org, id); err != nil {
		return err
	}
	event, err := credentialDeletionAuditEvent(ctx, org, id, "inbound")
	if err != nil {
		return apperror.NewInternal("attribute inbound credential deletion", err)
	}
	return writeAuthError(s.repo.ClearAuth(ctx, org, id, "inbound", event), "clear inbound carrier auth")
}

func credentialAuditEvent(ctx context.Context, organizationID, targetID uuid.UUID, direction, method string, rotation bool) (audit.Event, error) {
	action := "carrier.credential_set"
	if rotation {
		action = "carrier.credential_rotated"
	}
	actor, err := audit.ActorFromContext(ctx)
	if err != nil {
		return audit.Event{}, err
	}
	return audit.NewEvent(organizationID, actor, action, "carrier_connection", targetID, map[string]any{
		"direction": direction, "auth_method": method, "credential": "[REDACTED]",
	})
}

func credentialDeletionAuditEvent(ctx context.Context, organizationID, targetID uuid.UUID, direction string) (audit.Event, error) {
	actor, err := audit.ActorFromContext(ctx)
	if err != nil {
		return audit.Event{}, err
	}
	return audit.NewEvent(organizationID, actor, "carrier.credential_deleted", "carrier_connection", targetID, map[string]any{
		"direction": direction, "credential": "[REDACTED]",
	})
}

func requireAuthValue(value, name string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperror.NewBadRequest(name + " is required")
	}
	return value, nil
}
func writeAuthError(err error, message string) error {
	if err != nil {
		return apperror.NewInternal(message, err)
	}
	return nil
}

func digestHA1(username, realm, secret string) string {
	// SIP Digest with algorithm=MD5 requires the RFC-defined HA1 value.
	// codeql[go/weak-sensitive-data-hashing]
	return fmt.Sprintf("%x", md5.Sum([]byte(username+":"+realm+":"+secret)))
}

func digestWriteError(err error, message string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperror.NewConflict("digest username and realm are already in use")
	}
	return apperror.NewInternal(message, err)
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (Response, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return Response{}, err
	}
	if err := validateID(req.ProviderID, "provider_id"); err != nil {
		return Response{}, err
	}
	name, err := normalizeName(req.Name)
	if err != nil {
		return Response{}, err
	}
	if req.Status != nil {
		value, err := normalizeStatus(*req.Status)
		if err != nil {
			return Response{}, err
		}
		req.Status = &value
	}
	if err := validateLimits(req.MaxCPS, req.MaxConcurrentCalls, req.MaxDailyMinutes); err != nil {
		return Response{}, err
	}
	codecs, err := normalizeCodecs(req.Codecs)
	if err != nil {
		return Response{}, err
	}
	item, err := s.repo.Create(ctx, sqlc.CreateCarrierConnectionParams{
		OrganizationID:     &organizationID,
		ProviderID:         req.ProviderID,
		Name:               name,
		Status:             req.Status,
		InboundEnabled:     req.InboundEnabled,
		MaxCps:             req.MaxCPS,
		MaxConcurrentCalls: req.MaxConcurrentCalls,
		MaxDailyMinutes:    req.MaxDailyMinutes,
		Codecs:             codecs,
		SupportsVideo:      req.SupportsVideo,
		SupportsFax:        req.SupportsFax,
	})
	if err != nil {
		return Response{}, writeError(err, "carrier connection", "provider not found")
	}
	return response(item), nil
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Response, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return nil, err
	}
	items, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list carrier connections", err)
	}
	result := make([]Response, 0, len(items))
	for _, item := range items {
		result = append(result, listResponse(item))
	}
	return result, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Response, error) {
	if err := validateID(organizationID, "organization_id"); err != nil {
		return Response{}, err
	}
	if err := validateID(id, "carrier connection id"); err != nil {
		return Response{}, err
	}
	item, err := s.repo.Get(ctx, organizationID, id)
	if err != nil {
		return Response{}, readError(err, "carrier connection not found")
	}
	return getResponse(item), nil
}

func (s *Service) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req UpdateRequest,
) (Response, error) {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return Response{}, err
	}
	if req.Name == nil &&
		req.Status == nil &&
		req.InboundEnabled == nil &&
		req.MaxCPS == nil &&
		req.MaxConcurrentCalls == nil &&
		req.MaxDailyMinutes == nil &&
		req.Codecs == nil &&
		req.SupportsVideo == nil &&
		req.SupportsFax == nil {
		return Response{}, apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil {
		value, err := normalizeName(*req.Name)
		if err != nil {
			return Response{}, err
		}
		req.Name = &value
	}
	if req.Status != nil {
		value, err := normalizeStatus(*req.Status)
		if err != nil {
			return Response{}, err
		}
		req.Status = &value
	}
	if err := validateLimits(req.MaxCPS, req.MaxConcurrentCalls, req.MaxDailyMinutes); err != nil {
		return Response{}, err
	}
	codecs, err := normalizeCodecs(req.Codecs)
	if err != nil {
		return Response{}, err
	}
	item, err := s.repo.Update(ctx, sqlc.UpdateCarrierConnectionParams{
		Name:               req.Name,
		Status:             req.Status,
		InboundEnabled:     req.InboundEnabled,
		MaxCps:             req.MaxCPS,
		MaxConcurrentCalls: req.MaxConcurrentCalls,
		MaxDailyMinutes:    req.MaxDailyMinutes,
		Codecs:             codecs,
		SupportsVideo:      req.SupportsVideo,
		SupportsFax:        req.SupportsFax,
		ID:                 id,
		OrganizationID:     &organizationID,
	})
	if err != nil {
		return Response{}, writeError(err, "carrier connection", "carrier connection not found")
	}
	return response(item), nil
}

func (s *Service) Delete(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	if _, err := s.Get(ctx, organizationID, id); err != nil {
		return err
	}
	if err := s.repo.Disable(ctx, organizationID, id); err != nil {
		return apperror.NewInternal("disable carrier connection", err)
	}
	return nil
}

func (s *Service) CreateSourceIP(
	ctx context.Context,
	organizationID uuid.UUID,
	connectionID uuid.UUID,
	req SourceIPCreateRequest,
) (SourceIPResponse, error) {
	if _, err := s.Get(ctx, organizationID, connectionID); err != nil {
		return SourceIPResponse{}, err
	}
	cidr, err := normalizeCIDR(req.CIDR)
	if err != nil {
		return SourceIPResponse{}, err
	}
	item, err := s.repo.CreateSourceIP(ctx, organizationID, connectionID, cidr)
	if err != nil {
		return SourceIPResponse{}, writeError(err, "source IP", "carrier connection not found")
	}
	return sourceIPResponse(item), nil
}

func (s *Service) ListSourceIPs(
	ctx context.Context,
	organizationID uuid.UUID,
	connectionID uuid.UUID,
) ([]SourceIPResponse, error) {
	if _, err := s.Get(ctx, organizationID, connectionID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListSourceIPs(ctx, organizationID, connectionID)
	if err != nil {
		return nil, apperror.NewInternal("list carrier connection source IPs", err)
	}
	result := make([]SourceIPResponse, 0, len(items))
	for _, item := range items {
		result = append(result, sourceIPResponse(item))
	}
	return result, nil
}

func (s *Service) DeleteSourceIP(
	ctx context.Context,
	organizationID uuid.UUID,
	connectionID uuid.UUID,
	id uuid.UUID,
) error {
	if err := validateID(id, "source IP id"); err != nil {
		return err
	}
	items, err := s.ListSourceIPs(ctx, organizationID, connectionID)
	if err != nil {
		return err
	}
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
			break
		}
	}
	if !found {
		return apperror.NewNotFound("source IP not found")
	}
	if err := s.repo.DeleteSourceIP(ctx, organizationID, connectionID, id); err != nil {
		return apperror.NewInternal("delete source IP", err)
	}
	return nil
}

func (s *Service) ListProviders(ctx context.Context) ([]ProviderResponse, error) {
	providers, err := s.repo.ListProviders(ctx)
	if err != nil {
		return nil, apperror.NewInternal("list carrier providers", err)
	}

	result := make([]ProviderResponse, 0, len(providers))
	for _, provider := range providers {
		result = append(result, providerResponse(provider))
	}

	return result, nil
}

func (s *Service) GetProvider(ctx context.Context, id uuid.UUID) (ProviderResponse, error) {
	if err := validateID(id, "carrier provider id"); err != nil {
		return ProviderResponse{}, err
	}

	provider, err := s.repo.GetProvider(ctx, id)
	if err != nil {
		return ProviderResponse{}, readError(err, "carrier provider not found")
	}
	return providerResponse(provider), nil
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

package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo   *Repository
	cipher *encryption.Cipher
}

func NewService(repo *Repository, cipher *encryption.Cipher) *Service {
	return &Service{repo: repo, cipher: cipher}
}

func (s *Service) CreateCredential(ctx context.Context, organizationID uuid.UUID, req CreateCredentialRequest) (Credential, error) {
	if organizationID == uuid.Nil {
		return Credential{}, apperror.NewBadRequest("organization_id is required")
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.Name = strings.TrimSpace(req.Name)
	req.Secret = strings.TrimSpace(req.Secret)
	if !validProvider(req.Provider) {
		return Credential{}, apperror.NewBadRequest("unsupported AI provider")
	}
	if req.Name == "" || len(req.Name) > 128 {
		return Credential{}, apperror.NewBadRequest("credential name must be between 1 and 128 characters")
	}
	if req.Secret == "" {
		return Credential{}, apperror.NewBadRequest("provider secret is required")
	}
	if s.cipher == nil {
		return Credential{}, apperror.NewServiceUnavailable("AI provider credential encryption is unavailable", nil)
	}
	id := uuid.New()
	ciphertext, err := s.cipher.EncryptForScope(credentialScope(organizationID, id), req.Secret)
	if err != nil {
		return Credential{}, apperror.NewInternal("encrypt AI provider credential", err)
	}
	value, err := s.repo.CreateCredential(ctx, id, organizationID, req.Provider, req.Name, ciphertext)
	if conflict(err) {
		return Credential{}, apperror.NewConflict("AI provider credential already exists")
	}
	return value, dbError(err, "create AI provider credential")
}

func (s *Service) ListCredentials(ctx context.Context, organizationID uuid.UUID) ([]Credential, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.ListCredentials(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list AI provider credentials", err)
	}
	return values, nil
}

func (s *Service) RotateCredential(ctx context.Context, organizationID, id uuid.UUID, req RotateCredentialRequest) (Credential, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Credential{}, apperror.NewBadRequest("organization and credential ids are required")
	}
	req.Secret = strings.TrimSpace(req.Secret)
	if req.Secret == "" {
		return Credential{}, apperror.NewBadRequest("provider secret is required")
	}
	if s.cipher == nil {
		return Credential{}, apperror.NewServiceUnavailable("AI provider credential encryption is unavailable", nil)
	}
	if _, _, err := s.repo.GetCredentialCiphertext(ctx, organizationID, id); err != nil {
		return Credential{}, dbError(err, "AI provider credential not found")
	}
	ciphertext, err := s.cipher.EncryptForScope(credentialScope(organizationID, id), req.Secret)
	if err != nil {
		return Credential{}, apperror.NewInternal("encrypt AI provider credential", err)
	}
	value, err := s.repo.RotateCredential(ctx, organizationID, id, ciphertext)
	return value, dbError(err, "AI provider credential not found")
}

func (s *Service) DeleteCredential(ctx context.Context, organizationID, id uuid.UUID) error {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return apperror.NewBadRequest("organization and credential ids are required")
	}
	return dbError(s.repo.DeleteCredential(ctx, organizationID, id), "delete AI provider credential")
}

func (s *Service) UpsertBinding(ctx context.Context, organizationID, agentID uuid.UUID, role string, req UpsertBindingRequest) (Binding, error) {
	role = strings.TrimSpace(role)
	req.Provider = strings.TrimSpace(req.Provider)
	if organizationID == uuid.Nil || agentID == uuid.Nil {
		return Binding{}, apperror.NewBadRequest("organization and voice agent ids are required")
	}
	if !validRole(role) || !roleProviderCompatible(role, req.Provider) {
		return Binding{}, apperror.NewBadRequest("provider is not compatible with Voice Agent role")
	}
	if req.CredentialID == uuid.Nil {
		return Binding{}, apperror.NewBadRequest("credential_id is required")
	}
	if len(req.Config) == 0 {
		req.Config = json.RawMessage(`{}`)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(req.Config, &object) != nil || object == nil {
		return Binding{}, apperror.NewBadRequest("provider config must be a JSON object")
	}
	credential, _, err := s.repo.GetCredentialCiphertext(ctx, organizationID, req.CredentialID)
	if err != nil {
		return Binding{}, dbError(err, "AI provider credential not found")
	}
	if credential.Provider != req.Provider {
		return Binding{}, apperror.NewBadRequest("credential provider does not match binding provider")
	}
	value, err := s.repo.UpsertBinding(ctx, organizationID, agentID, role, req)
	return value, dbError(err, "bind AI provider")
}

func (s *Service) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]Binding, error) {
	values, err := s.repo.ListBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal("list Voice Agent provider bindings", err)
	}
	return values, nil
}

func (s *Service) DeleteBinding(ctx context.Context, organizationID, agentID uuid.UUID, role string) error {
	if !validRole(role) {
		return apperror.NewBadRequest("invalid provider role")
	}
	return dbError(s.repo.DeleteBinding(ctx, organizationID, agentID, role), "delete Voice Agent provider binding")
}

func (s *Service) Resolve(
	ctx context.Context,
	organizationID, agentID uuid.UUID,
) ([]ResolvedBinding, error) {
	rows, err := s.repo.ResolveBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal("resolve Voice Agent provider bindings", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if s.cipher == nil {
		return nil, apperror.NewServiceUnavailable(
			"AI provider credential encryption is unavailable",
			nil,
		)
	}

	out := make([]ResolvedBinding, 0, len(rows))
	for _, row := range rows {
		secret, decryptErr := s.cipher.DecryptForScope(credentialScope(organizationID, row.CredentialID), row.SecretCiphertext)
		if decryptErr != nil {
			return nil, apperror.NewInternal("decrypt AI provider credential", decryptErr)
		}
		out = append(out, ResolvedBinding{
			Role:     row.Role,
			Provider: row.Provider,
			APIKey:   secret,
			Config:   append(json.RawMessage(nil), row.Config...),
		})
	}
	return out, nil
}

func credentialScope(organizationID, credentialID uuid.UUID) string {
	return fmt.Sprintf("ai-provider:%s:%s", organizationID, credentialID)
}

func validProvider(value string) bool {
	switch value {
	case ProviderOpenAI, ProviderDeepgram, ProviderGroq, ProviderCartesia:
		return true
	default:
		return false
	}
}

func validRole(value string) bool {
	switch value {
	case RoleRealtime, RoleSTT, RoleLLM, RoleTTS:
		return true
	default:
		return false
	}
}

func roleProviderCompatible(role, provider string) bool {
	switch role {
	case RoleRealtime:
		return provider == ProviderOpenAI
	case RoleSTT:
		return provider == ProviderDeepgram
	case RoleLLM:
		return provider == ProviderGroq
	case RoleTTS:
		return provider == ProviderCartesia
	default:
		return false
	}
}

func dbError(err error, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}

func conflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

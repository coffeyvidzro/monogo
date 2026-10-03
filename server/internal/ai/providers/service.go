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
	repo     *Repository
	cipher   *encryption.Cipher
	verifier *Verifier
}

func NewService(
	repo *Repository,
	cipher *encryption.Cipher,
	verifiers ...*Verifier,
) *Service {
	verifier := NewVerifier(nil)
	if len(verifiers) > 0 && verifiers[0] != nil {
		verifier = verifiers[0]
	}
	return &Service{
		repo:     repo,
		cipher:   cipher,
		verifier: verifier,
	}
}

func (s *Service) CreateCredential(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateCredentialRequest,
) (Credential, error) {
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
		return Credential{}, apperror.NewBadRequest(
			"credential name must be between 1 and 128 characters",
		)
	}
	if req.Secret == "" {
		return Credential{}, apperror.NewBadRequest("provider secret is required")
	}
	if s.cipher == nil {
		return Credential{}, apperror.NewServiceUnavailable(
			"AI provider credential encryption is unavailable",
			nil,
		)
	}

	id := uuid.New()
	ciphertext, err := s.cipher.EncryptForScope(
		credentialScope(organizationID, id),
		req.Secret,
	)
	if err != nil {
		return Credential{}, apperror.NewInternal(
			"encrypt AI provider credential",
			err,
		)
	}

	value, err := s.repo.CreateCredential(
		ctx,
		id,
		organizationID,
		req.Provider,
		req.Name,
		ciphertext,
	)
	if conflict(err) {
		return Credential{}, apperror.NewConflict(
			"AI provider credential already exists",
		)
	}
	if err != nil {
		return Credential{}, dbError(err, "create AI provider credential")
	}

	return value, nil
}

func (s *Service) ListCredentials(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Credential, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}

	values, err := s.repo.ListCredentials(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal(
			"list AI provider credentials",
			err,
		)
	}

	return values, nil
}

func (s *Service) ListIntegrations(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Integration, error) {
	credentials, err := s.ListCredentials(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]Integration, 0, len(credentials))
	for _, credential := range credentials {
		voiceAgentIDs, err := s.repo.ListVoiceAgentIDs(
			ctx,
			organizationID,
			credential.ID,
		)
		if err != nil {
			return nil, apperror.NewInternal(
				"list AI integration usage",
				err,
			)
		}
		result = append(result, Integration{
			Credential:    credential,
			VoiceAgentIDs: voiceAgentIDs,
		})
	}
	return result, nil
}

func (s *Service) VerifyIntegration(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Credential, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Credential{}, apperror.NewBadRequest(
			"organization and integration ids are required",
		)
	}
	credential, ciphertext, err := s.repo.GetCredentialCiphertext(
		ctx,
		organizationID,
		id,
	)
	if err != nil {
		return Credential{}, dbError(err, "AI integration not found")
	}
	if s.cipher == nil {
		return Credential{}, apperror.NewServiceUnavailable(
			"AI provider credential encryption is unavailable",
			nil,
		)
	}
	secret, err := s.cipher.DecryptForScope(
		credentialScope(organizationID, id),
		ciphertext,
	)
	if err != nil {
		return Credential{}, apperror.NewInternal(
			"decrypt AI provider credential",
			err,
		)
	}
	result, err := s.verifier.Verify(ctx, credential.Provider, secret)
	if err != nil {
		return Credential{}, apperror.NewInternal(
			"verify AI integration",
			err,
		)
	}
	var failureCode *string
	if result.FailureCode != "" {
		failureCode = &result.FailureCode
	}
	value, err := s.repo.UpdateVerification(
		ctx,
		organizationID,
		id,
		result.State,
		failureCode,
	)
	if err != nil {
		return Credential{}, dbError(err, "update AI integration verification")
	}
	return value, nil
}

func (s *Service) RotateCredential(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req RotateCredentialRequest,
) (Credential, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Credential{}, apperror.NewBadRequest(
			"organization and credential ids are required",
		)
	}

	req.Secret = strings.TrimSpace(req.Secret)
	if req.Secret == "" {
		return Credential{}, apperror.NewBadRequest("provider secret is required")
	}
	if s.cipher == nil {
		return Credential{}, apperror.NewServiceUnavailable(
			"AI provider credential encryption is unavailable",
			nil,
		)
	}

	if _, _, err := s.repo.GetCredentialCiphertext(
		ctx,
		organizationID,
		id,
	); err != nil {
		return Credential{}, dbError(err, "AI provider credential not found")
	}

	ciphertext, err := s.cipher.EncryptForScope(
		credentialScope(organizationID, id),
		req.Secret,
	)
	if err != nil {
		return Credential{}, apperror.NewInternal(
			"encrypt AI provider credential",
			err,
		)
	}

	value, err := s.repo.RotateCredential(
		ctx,
		organizationID,
		id,
		ciphertext,
	)
	if err != nil {
		return Credential{}, dbError(err, "AI provider credential not found")
	}

	return value, nil
}

func (s *Service) DeleteCredential(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return apperror.NewBadRequest(
			"organization and credential ids are required",
		)
	}

	return dbError(
		s.repo.DeleteCredential(ctx, organizationID, id),
		"delete AI provider credential",
	)
}

func (s *Service) UpsertBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
	role string,
	req UpsertBindingRequest,
) (Binding, error) {
	role = strings.TrimSpace(role)
	req.Provider = strings.TrimSpace(req.Provider)

	if organizationID == uuid.Nil || agentID == uuid.Nil {
		return Binding{}, apperror.NewBadRequest(
			"organization and voice agent ids are required",
		)
	}
	if !validRole(role) || !roleProviderCompatible(role, req.Provider) {
		return Binding{}, apperror.NewBadRequest(
			"provider is not compatible with Voice Agent role",
		)
	}
	if req.CredentialID == uuid.Nil {
		return Binding{}, apperror.NewBadRequest("credential_id is required")
	}

	if len(req.Config) == 0 {
		req.Config = json.RawMessage(`{}`)
	}

	if err := validateProviderConfig(req.Provider, req.Config); err != nil {
		return Binding{}, err
	}

	credential, _, err := s.repo.GetCredentialCiphertext(
		ctx,
		organizationID,
		req.CredentialID,
	)
	if err != nil {
		return Binding{}, dbError(err, "AI provider credential not found")
	}
	if credential.Provider != req.Provider {
		return Binding{}, apperror.NewBadRequest(
			"credential provider does not match binding provider",
		)
	}

	value, err := s.repo.UpsertBinding(
		ctx,
		organizationID,
		agentID,
		role,
		req,
	)
	if err != nil {
		return Binding{}, dbError(err, "bind AI provider")
	}

	return value, nil
}

func (s *Service) BindingStatuses(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) ([]BindingStatus, error) {
	bindings, err := s.ListBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, err
	}
	result := make([]BindingStatus, 0, len(bindings))
	for _, binding := range bindings {
		credential, _, err := s.repo.GetCredentialCiphertext(
			ctx,
			organizationID,
			binding.CredentialID,
		)
		if err != nil {
			return nil, dbError(err, "AI integration not found")
		}
		result = append(result, BindingStatus{
			Role:            binding.Role,
			Provider:        binding.Provider,
			IntegrationID:   binding.CredentialID,
			ConnectionState: credential.ConnectionState,
			FailureCode:     credential.FailureCode,
			Config:          append(json.RawMessage(nil), binding.Config...),
		})
	}
	return result, nil
}

func validateProviderConfig(provider string, value json.RawMessage) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(value, &object); err != nil || object == nil {
		return apperror.NewBadRequest("provider config must be a JSON object")
	}
	allowed := map[string]string{}
	switch provider {
	case ProviderDeepgram:
		allowed = map[string]string{
			"model":    "string",
			"language": "string",
		}
	case ProviderGroq:
		allowed = map[string]string{
			"model":       "string",
			"temperature": "number",
		}
	case ProviderCartesia:
		allowed = map[string]string{
			"model":    "string",
			"voice_id": "string",
			"language": "string",
		}
	case ProviderOpenAI:
		allowed = map[string]string{
			"model": "string",
			"voice": "string",
		}
	default:
		return apperror.NewBadRequest("unsupported AI provider")
	}
	for key, raw := range object {
		kind, ok := allowed[key]
		if !ok {
			return apperror.NewBadRequest("unsupported provider config field: " + key)
		}
		var target any
		if kind == "string" {
			target = new(string)
		} else {
			target = new(float64)
		}
		if err := json.Unmarshal(raw, target); err != nil {
			return apperror.NewBadRequest("invalid provider config field: " + key)
		}
	}
	return nil
}

func (s *Service) ListBindings(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) ([]Binding, error) {
	values, err := s.repo.ListBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal(
			"list Voice Agent provider bindings",
			err,
		)
	}

	return values, nil
}

func (s *Service) DeleteBinding(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
	role string,
) error {
	if !validRole(role) {
		return apperror.NewBadRequest("invalid provider role")
	}

	return dbError(
		s.repo.DeleteBinding(ctx, organizationID, agentID, role),
		"delete Voice Agent provider binding",
	)
}

func (s *Service) Resolve(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) ([]ResolvedBinding, error) {
	rows, err := s.repo.ResolveBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal(
			"resolve Voice Agent provider bindings",
			err,
		)
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

	result := make([]ResolvedBinding, 0, len(rows))
	for _, row := range rows {
		secret, err := s.cipher.DecryptForScope(
			credentialScope(organizationID, row.CredentialID),
			row.SecretCiphertext,
		)
		if err != nil {
			return nil, apperror.NewInternal(
				"decrypt AI provider credential",
				err,
			)
		}

		result = append(result, ResolvedBinding{
			Role:     row.Role,
			Provider: row.Provider,
			APIKey:   secret,
			Config:   append(json.RawMessage(nil), row.Config...),
		})
	}

	return result, nil
}

func (s *Service) ResolveSnapshot(
	ctx context.Context,
	organizationID uuid.UUID,
	value json.RawMessage,
) ([]ResolvedBinding, error) {
	var bindings []struct {
		Role         string          `json:"role"`
		Provider     string          `json:"provider"`
		CredentialID uuid.UUID       `json:"credential_id"`
		Config       json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(value, &bindings); err != nil {
		return nil, apperror.NewInternal("decode provider binding snapshot", err)
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	if s.cipher == nil {
		return nil, apperror.NewServiceUnavailable(
			"AI provider credential encryption is unavailable",
			nil,
		)
	}
	result := make([]ResolvedBinding, 0, len(bindings))
	for _, binding := range bindings {
		credential, ciphertext, err := s.repo.GetCredentialCiphertext(
			ctx,
			organizationID,
			binding.CredentialID,
		)
		if err != nil {
			return nil, dbError(err, "snapshotted AI integration not found")
		}
		if credential.Provider != binding.Provider {
			return nil, apperror.NewInternal(
				"snapshotted AI integration provider mismatch",
				nil,
			)
		}
		secret, err := s.cipher.DecryptForScope(
			credentialScope(organizationID, binding.CredentialID),
			ciphertext,
		)
		if err != nil {
			return nil, apperror.NewInternal(
				"decrypt snapshotted AI integration",
				err,
			)
		}
		result = append(result, ResolvedBinding{
			Role:     binding.Role,
			Provider: binding.Provider,
			APIKey:   secret,
			Config:   append(json.RawMessage(nil), binding.Config...),
		})
	}
	return result, nil
}

func credentialScope(
	organizationID uuid.UUID,
	credentialID uuid.UUID,
) string {
	return fmt.Sprintf(
		"ai-provider:%s:%s",
		organizationID,
		credentialID,
	)
}

func validProvider(value string) bool {
	switch value {
	case ProviderOpenAI,
		ProviderDeepgram,
		ProviderGroq,
		ProviderCartesia:
		return true
	default:
		return false
	}
}

func validRole(value string) bool {
	switch value {
	case RoleRealtime,
		RoleSTT,
		RoleLLM,
		RoleTTS:
		return true
	default:
		return false
	}
}

func roleProviderCompatible(role string, provider string) bool {
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

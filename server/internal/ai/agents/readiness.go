package agents

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/ai/providers"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func (s *Service) Readiness(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) (ReadinessReport, error) {
	agent, err := s.Get(ctx, organizationID, agentID)
	if err != nil {
		return ReadinessReport{}, err
	}
	report := ReadinessReport{
		ConfigurationRevision: agent.ConfigurationRevision,
		ActiveRevision:        agent.ActiveRevision,
		Engine:                agent.Engine,
		Preset:                agent.Preset,
		PresetVersion:         agent.PresetVersion,
		Bindings:              []BindingDiagnostic{},
		Issues:                []ReadinessIssue{},
	}
	if s.providers == nil {
		report.Issues = append(report.Issues, readinessIssue(
			"provider_service_unavailable",
			"bindings",
			"AI provider readiness is unavailable.",
			"Restore the AI provider service before activating the agent.",
		))
		return report, nil
	}
	statuses, err := s.providers.BindingStatuses(ctx, organizationID, agentID)
	if err != nil {
		return ReadinessReport{}, err
	}
	byRole := make(map[string]providers.BindingStatus, len(statuses))
	for _, status := range statuses {
		byRole[status.Role] = status
		report.Bindings = append(report.Bindings, BindingDiagnostic{
			Role:            status.Role,
			Provider:        status.Provider,
			IntegrationID:   status.IntegrationID,
			ConnectionState: status.ConnectionState,
			FailureCode:     status.FailureCode,
			Config:          append([]byte(nil), status.Config...),
		})
	}
	expected := expectedProviders(agent.Engine)
	for _, status := range statuses {
		if _, ok := expected[status.Role]; !ok {
			report.Issues = append(report.Issues, readinessIssue(
				"unexpected_provider_binding",
				"bindings."+status.Role,
				fmt.Sprintf("The %s role is not valid for the selected engine.", status.Role),
				"Remove the binding or select the matching engine mode.",
			))
		}
	}
	for _, role := range []string{
		providers.RoleRealtime,
		providers.RoleSTT,
		providers.RoleLLM,
		providers.RoleTTS,
	} {
		provider, expectedRole := expected[role]
		if !expectedRole {
			continue
		}
		status, ok := byRole[role]
		if !ok || status.Provider != provider {
			report.Issues = append(report.Issues, readinessIssue(
				"missing_provider_binding",
				"bindings."+role,
				fmt.Sprintf("The %s role requires a %s integration.", role, provider),
				fmt.Sprintf("Connect a ready %s integration to the %s role.", provider, role),
			))
			continue
		}
		if status.ConnectionState != providers.ConnectionReady {
			report.Issues = append(report.Issues, readinessIssue(
				"integration_not_ready",
				"bindings."+role+".integration_id",
				fmt.Sprintf("The %s integration is not ready.", provider),
				"Verify or rotate the integration, then run readiness again.",
			))
		}
	}
	report.Ready = len(report.Issues) == 0
	return report, nil
}

func (s *Service) Activate(
	ctx context.Context,
	organizationID uuid.UUID,
	agentID uuid.UUID,
) (sqlc.VoiceAgent, ReadinessReport, error) {
	report, err := s.Readiness(ctx, organizationID, agentID)
	if err != nil {
		return sqlc.VoiceAgent{}, ReadinessReport{}, err
	}
	if !report.Ready {
		return sqlc.VoiceAgent{}, report, apperror.NewConflict(
			"Voice Agent is not ready for activation",
		)
	}
	agent, err := s.repo.Activate(ctx, organizationID, agentID)
	if err != nil {
		return sqlc.VoiceAgent{}, report, writeError(
			err,
			"Voice Agent readiness changed during activation",
		)
	}
	report.ActiveRevision = agent.ActiveRevision
	return agent, report, nil
}

func expectedProviders(engine string) map[string]string {
	if engine == EngineIntegrated {
		return map[string]string{
			providers.RoleRealtime: providers.ProviderOpenAI,
		}
	}
	return map[string]string{
		providers.RoleSTT: providers.ProviderDeepgram,
		providers.RoleLLM: providers.ProviderGroq,
		providers.RoleTTS: providers.ProviderCartesia,
	}
}

func readinessIssue(
	code string,
	field string,
	message string,
	remediation string,
) ReadinessIssue {
	return ReadinessIssue{
		Code:        code,
		Field:       field,
		Message:     message,
		Remediation: remediation,
	}
}

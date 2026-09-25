package didww

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/telecom/lifecycle"
	"github.com/google/uuid"
)

type LifecycleProvider struct{ client *Client }

func NewLifecycleProvider(client *Client) *LifecycleProvider {
	return &LifecycleProvider{client: client}
}

func (p *LifecycleProvider) RequestRelease(ctx context.Context, resourceID string) (string, error) {
	did, err := p.client.TerminateDID(ctx, resourceID)
	return did.ID, err
}

func (p *LifecycleProvider) ReleaseCompleted(ctx context.Context, resourceID string) (bool, error) {
	did, err := p.client.GetDID(ctx, resourceID)
	if err != nil {
		return false, err
	}
	return did.Attributes.Terminated, nil
}

// DIDWW emergency and porting APIs require account-specific regulatory
// products. They remain explicit capabilities rather than guessed endpoints;
// deployments enable them by supplying an adapter implementing this boundary.
func (*LifecycleProvider) ValidateEmergency(context.Context, string, lifecycle.EmergencyAddressRequest) (string, bool, string, error) {
	return "", false, "", lifecycle.ErrProviderCapabilityUnavailable
}

func (*LifecycleProvider) SubmitPortIn(context.Context, uuid.UUID, lifecycle.PortInRequest, []sqlc.PortInDocument) (string, error) {
	return "", lifecycle.ErrProviderCapabilityUnavailable
}

func (*LifecycleProvider) CheckPortability(context.Context, string) (bool, string, error) {
	return false, "", lifecycle.ErrProviderCapabilityUnavailable
}

func (*LifecycleProvider) PortInStatus(context.Context, string) (lifecycle.ProviderPortStatus, error) {
	return lifecycle.ProviderPortStatus{}, lifecycle.ErrProviderCapabilityUnavailable
}

package lifecycle

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/google/uuid"
)

var ErrProviderCapabilityUnavailable = errors.New("provider lifecycle capability is unavailable")

type DIDWWLifecycleProvider struct{ client *didww.Client }

func NewDIDWWLifecycleProvider(client *didww.Client) *DIDWWLifecycleProvider {
	return &DIDWWLifecycleProvider{client: client}
}

func (p *DIDWWLifecycleProvider) RequestRelease(ctx context.Context, resourceID string) (string, error) {
	did, err := p.client.TerminateDID(ctx, resourceID)
	return did.ID, err
}

func (p *DIDWWLifecycleProvider) ReleaseCompleted(ctx context.Context, resourceID string) (bool, error) {
	did, err := p.client.GetDID(ctx, resourceID)
	if err != nil {
		return false, err
	}
	return did.Attributes.Terminated, nil
}

// DIDWW emergency and porting APIs require account-specific regulatory
// products. They remain explicit capabilities rather than guessed endpoints;
// deployments enable them by supplying an adapter implementing this boundary.
func (*DIDWWLifecycleProvider) ValidateEmergency(context.Context, string, EmergencyAddressRequest) (string, bool, string, error) {
	return "", false, "", ErrProviderCapabilityUnavailable
}

func (*DIDWWLifecycleProvider) SubmitPortIn(context.Context, uuid.UUID, PortInRequest, []sqlc.PortInDocument) (string, error) {
	return "", ErrProviderCapabilityUnavailable
}

func (*DIDWWLifecycleProvider) CheckPortability(context.Context, string) (bool, string, error) {
	return false, "", ErrProviderCapabilityUnavailable
}

func (*DIDWWLifecycleProvider) PortInStatus(context.Context, string) (ProviderPortStatus, error) {
	return ProviderPortStatus{}, ErrProviderCapabilityUnavailable
}

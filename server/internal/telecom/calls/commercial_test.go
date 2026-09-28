package calls

import (
	"context"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/commercial/authorization"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
)

type commercialAuthorizerSpy struct {
	requests []authorization.ManagedCallRequest
	err      error
}

func (s *commercialAuthorizerSpy) AuthorizeManagedCall(
	_ context.Context,
	req authorization.ManagedCallRequest,
) (authorization.CallAuthorization, error) {
	s.requests = append(s.requests, req)
	return authorization.CallAuthorization{}, s.err
}

func TestAuthorizeOutboundUsesTrustedRouteProvisioningMode(t *testing.T) {
	spy := &commercialAuthorizerSpy{}
	service := &Service{
		commercial: spy,
	}
	callID := uuid.New()
	organizationID := uuid.New()

	err := service.authorizeOutbound(
		context.Background(),
		callID,
		organizationID,
		"+14155550100",
		routing.OutboundRoute{
			ProvisioningMode: "managed",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(spy.requests) != 1 {
		t.Fatalf("authorization requests = %d, want 1", len(spy.requests))
	}
	request := spy.requests[0]
	if request.CallID != callID || request.OrganizationID != organizationID || request.Destination != "+14155550100" {
		t.Fatalf("unexpected authorization request: %+v", request)
	}

	err = service.authorizeOutbound(
		context.Background(),
		callID,
		organizationID,
		"+14155550100",
		routing.OutboundRoute{
			ProvisioningMode: "byoc",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(spy.requests) != 1 {
		t.Fatalf("BYOC invoked commercial authorization; requests = %d", len(spy.requests))
	}
}

func TestAuthorizeOutboundRejectsUnknownProvisioningMode(t *testing.T) {
	service := &Service{
		commercial: &commercialAuthorizerSpy{},
	}
	err := service.authorizeOutbound(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"+14155550100",
		routing.OutboundRoute{
			ProvisioningMode: "",
		},
	)
	if err == nil {
		t.Fatal("expected unknown provisioning mode to fail closed")
	}
}

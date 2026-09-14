package calls

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

const (
	openSIPSEgressHost         = "opensips"
	openSIPSEgressPort         = 5060
	routeURIHeaderVar          = "sip_h_X-Leamout-Route-URI"
	carrierConnectionHeaderVar = "sip_h_X-Leamout-Carrier-Connection-ID"
	privacyHeaderVar           = "sip_h_X-Leamout-Privacy"
	dtmfTypeVar                = "dtmf_type"
	codecStringVar             = "absolute_codec_string"
	mediaEncryptionHeaderVar   = "sip_h_X-Leamout-Media-Encryption"
)

// FreeSWITCHController adapts the FreeSWITCH client to the calls.Controller interface.
// It translates resolved domain-level call operations into FreeSWITCH-specific commands.
type FreeSWITCHController struct {
	client *freeswitch.Client
}

var _ Controller = (*FreeSWITCHController)(nil)

// NewFreeSWITCHController creates a new adapter wrapping the FreeSWITCH client.
func NewFreeSWITCHController(client *freeswitch.Client) *FreeSWITCHController {
	if client == nil {
		panic("calls: FreeSWITCH client is required")
	}

	return &FreeSWITCHController{client: client}
}

// Originate initiates an outbound call through the trusted FreeSWITCH -> OpenSIPS
// handoff. The carrier destination is carried as internal SIP metadata and is
// never exposed as a public FreeSWITCH dial string.
func (c *FreeSWITCHController) Originate(ctx context.Context, req OriginateRequest) (string, error) {
	endpoint, routeURI, err := freeSWITCHEgress(req)
	if err != nil {
		return "", err
	}

	variables, err := egressVariables(req, routeURI)
	if err != nil {
		return "", err
	}

	call, err := c.client.Originate(ctx, freeswitch.OriginateRequest{
		Endpoint:    endpoint,
		Destination: req.Destination,
		CallerID:    req.CallerID,
		Variables:   variables,
	})
	if err != nil {
		return "", fmt.Errorf("originate call: %w", err)
	}
	if call.UUID == "" {
		return "", fmt.Errorf("FreeSWITCH returned empty call UUID")
	}

	return call.UUID, nil
}

func egressVariables(req OriginateRequest, routeURI string) (map[string]string, error) {
	if req.CarrierConnectionID == uuid.Nil {
		return nil, fmt.Errorf("resolved carrier connection id is required")
	}

	variables := make(map[string]string, len(req.Variables)+5)
	for key, value := range req.Variables {
		variables[key] = value
	}
	// Always overwrite reserved metadata after copying user variables.
	variables[routeURIHeaderVar] = routeURI
	variables[carrierConnectionHeaderVar] = req.CarrierConnectionID.String()
	if req.Privacy {
		variables[privacyHeaderVar] = "id"
	}
	variables[dtmfTypeVar] = req.DTMFMode
	variables[codecStringVar] = strings.Join(req.Codecs, ",")
	variables[mediaEncryptionHeaderVar] = req.MediaEncryption
	return variables, nil
}

func freeSWITCHEgress(req OriginateRequest) (string, string, error) {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		return "", "", fmt.Errorf("resolved route host is required")
	}
	if req.Port < 1 || req.Port > 65535 {
		return "", "", fmt.Errorf("resolved route port is invalid: %d", req.Port)
	}

	transport := strings.ToLower(strings.TrimSpace(req.Transport))
	switch transport {
	case "udp", "tcp", "tls":
	default:
		return "", "", fmt.Errorf("resolved route transport is invalid: %q", req.Transport)
	}

	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		return "", "", fmt.Errorf("resolved route destination is required")
	}

	carrierTarget := net.JoinHostPort(host, strconv.Itoa(int(req.Port)))
	routeURI := fmt.Sprintf("sip:%s;transport=%s", carrierTarget, transport)
	openSIPSTarget := net.JoinHostPort(openSIPSEgressHost, strconv.Itoa(openSIPSEgressPort))
	endpoint := fmt.Sprintf("sofia/internal/%s@%s;transport=udp", destination, openSIPSTarget)

	return endpoint, routeURI, nil
}

func (c *FreeSWITCHController) Answer(ctx context.Context, callID string) error {
	if err := c.client.Answer(ctx, callID); err != nil {
		return fmt.Errorf("answer call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Hangup(ctx context.Context, callID string) error {
	if err := c.client.Hangup(ctx, callID); err != nil {
		return fmt.Errorf("hangup call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Transfer(ctx context.Context, callID string, req TransferRequest) error {
	if err := c.client.Transfer(ctx, freeswitch.TransferRequest{
		CallID:      callID,
		Destination: req.Destination,
		Dialplan:    req.Dialplan,
		Context:     req.Context,
	}); err != nil {
		return fmt.Errorf("transfer call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Hold(ctx context.Context, callID string) error {
	if err := c.client.Hold(ctx, callID); err != nil {
		return fmt.Errorf("hold call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Unhold(ctx context.Context, callID string) error {
	if err := c.client.Unhold(ctx, callID); err != nil {
		return fmt.Errorf("unhold call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Play(ctx context.Context, callID, path string) error {
	if err := c.client.PlayAudio(ctx, callID, path); err != nil {
		return fmt.Errorf("play audio: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Stop(ctx context.Context, callID string) error {
	if err := c.client.StopAudio(ctx, callID); err != nil {
		return fmt.Errorf("stop audio: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Record(ctx context.Context, callID string, req RecordRequest) error {
	if err := c.client.Record(ctx, freeswitch.RecordRequest{
		CallID: callID,
		Path:   req.Path,
		Action: req.Action,
	}); err != nil {
		return fmt.Errorf("record call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) DTMF(ctx context.Context, callID, digits string) error {
	if err := c.client.SendDTMF(ctx, callID, digits); err != nil {
		return fmt.Errorf("send DTMF: %w", err)
	}
	return nil
}

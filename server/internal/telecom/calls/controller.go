package calls

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/google/uuid"
)

const (
	openSIPSEgressHost         = "opensips"
	openSIPSEgressPort         = 5060
	leamoutCallIDVar            = "leamout_call_id"
	routeURIHeaderVar           = "sip_h_X-Leamout-Route-URI"
	carrierConnectionHeaderVar = "sip_h_X-Leamout-Carrier-Connection-ID"
	privacyHeaderVar            = "sip_h_X-Leamout-Privacy"
	dtmfTypeVar                 = "dtmf_type"
	mediaEncryptionHeaderVar   = "sip_h_X-Leamout-Media-Encryption"
)

// Controller is the media-server contract used by call controls.
type Controller interface {
	Originate(context.Context, OriginateRequest) (OriginateResult, error)
	Answer(context.Context, string) error
	Hangup(context.Context, string) error
	Transfer(context.Context, string, TransferRequest) error
	Hold(context.Context, string) error
	Resume(context.Context, string) error
	PlayAudio(context.Context, string, string) error
	StopPlayback(context.Context, string) error
	Record(context.Context, string, RecordRequest) error
	SendDTMF(context.Context, string, string) error
	SetCallID(context.Context, string, uuid.UUID) error
}

type OriginateRequest struct {
	CallID              uuid.UUID
	Destination         string
	CallerID            string
	CarrierConnectionID uuid.UUID
	Host                string
	Port                uint16
	Transport           string
	Privacy             bool
	DTMFMode            string
	MediaEncryption     string
}

type OriginateResult struct {
	ChannelID string
}

type TransferRequest struct {
	Destination string
	Dialplan    string
	Context     string
}

type RecordRequest struct {
	Path   string
	Action string
}

// FreeSWITCHController adapts the FreeSWITCH client to call controls.
type FreeSWITCHController struct {
	client *freeswitch.Client
}

var _ Controller = (*FreeSWITCHController)(nil)

func NewFreeSWITCHController(client *freeswitch.Client) *FreeSWITCHController {
	if client == nil {
		panic("calls: FreeSWITCH client is required")
	}
	return &FreeSWITCHController{client: client}
}

func (c *FreeSWITCHController) Originate(
	ctx context.Context,
	req OriginateRequest,
) (OriginateResult, error) {
	endpoint, routeURI, err := freeSWITCHEgress(req)
	if err != nil {
		return OriginateResult{}, err
	}

	variables, err := egressVariables(req, routeURI)
	if err != nil {
		return OriginateResult{}, err
	}

	call, err := c.client.Originate(ctx, freeswitch.OriginateRequest{
		Endpoint:    endpoint,
		Destination: req.Destination,
		CallerID:    req.CallerID,
		Variables:   variables,
	})
	if err != nil {
		return OriginateResult{}, fmt.Errorf("originate call: %w", err)
	}
	if strings.TrimSpace(call.UUID) == "" {
		return OriginateResult{}, fmt.Errorf("FreeSWITCH returned empty channel UUID")
	}

	return OriginateResult{ChannelID: call.UUID}, nil
}

func egressVariables(req OriginateRequest, routeURI string) (map[string]string, error) {
	if req.CallID == uuid.Nil {
		return nil, fmt.Errorf("call id is required")
	}
	if req.CarrierConnectionID == uuid.Nil {
		return nil, fmt.Errorf("resolved carrier connection id is required")
	}

	variables := map[string]string{
		leamoutCallIDVar:            req.CallID.String(),
		routeURIHeaderVar:           routeURI,
		carrierConnectionHeaderVar: req.CarrierConnectionID.String(),
	}

	if req.Privacy {
		variables[privacyHeaderVar] = "id"
	}

	if mode := strings.ToLower(strings.TrimSpace(req.DTMFMode)); mode != "" {
		switch mode {
		case "rfc2833", "info", "none":
			variables[dtmfTypeVar] = mode
		default:
			return nil, fmt.Errorf("DTMF mode is invalid: %q", req.DTMFMode)
		}
	}

	if encryption := strings.ToLower(strings.TrimSpace(req.MediaEncryption)); encryption != "" {
		switch encryption {
		case "none", "sdes_srtp":
			variables[mediaEncryptionHeaderVar] = encryption
		default:
			return nil, fmt.Errorf("media encryption is invalid: %q", req.MediaEncryption)
		}
	}

	return variables, nil
}

func freeSWITCHEgress(req OriginateRequest) (string, string, error) {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		return "", "", fmt.Errorf("resolved route host is required")
	}
	if strings.ContainsAny(host, " 	
,{}[]") {
		return "", "", fmt.Errorf("resolved route host is invalid")
	}
	if req.Port == 0 {
		return "", "", fmt.Errorf("resolved route port is required")
	}

	transport := strings.ToLower(strings.TrimSpace(req.Transport))
	switch transport {
	case "udp", "tcp", "tls":
	default:
		return "", "", fmt.Errorf("resolved route transport is invalid: %q", req.Transport)
	}

	destination := strings.TrimSpace(req.Destination)
	if !validPSTNAddress(destination) {
		return "", "", fmt.Errorf("resolved route destination is invalid")
	}

	callerID := strings.TrimSpace(req.CallerID)
	if callerID != "" && !validPSTNAddress(callerID) {
		return "", "", fmt.Errorf("caller id is invalid")
	}

	carrierTarget := net.JoinHostPort(host, strconv.Itoa(int(req.Port)))
	routeURI := fmt.Sprintf("sip:%s;transport=%s", carrierTarget, transport)

	openSIPSTarget := net.JoinHostPort(openSIPSEgressHost, strconv.Itoa(openSIPSEgressPort))
	endpoint := fmt.Sprintf("sofia/internal/%s@%s;transport=udp", destination, openSIPSTarget)

	return endpoint, routeURI, nil
}

func validPSTNAddress(value string) bool {
	if value == "" || len(value) > 32 {
		return false
	}
	for i, r := range value {
		if i == 0 && r == '+' {
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return value != "+"
}

func (c *FreeSWITCHController) Answer(ctx context.Context, channelID string) error {
	if err := c.client.Answer(ctx, channelID); err != nil {
		return fmt.Errorf("answer call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Hangup(ctx context.Context, channelID string) error {
	if err := c.client.Hangup(ctx, channelID); err != nil {
		return fmt.Errorf("hangup call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Transfer(
	ctx context.Context,
	channelID string,
	req TransferRequest,
) error {
	if err := c.client.Transfer(ctx, freeswitch.TransferRequest{
		CallID:      channelID,
		Destination: req.Destination,
		Dialplan:    req.Dialplan,
		Context:     req.Context,
	}); err != nil {
		return fmt.Errorf("transfer call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Hold(ctx context.Context, channelID string) error {
	if err := c.client.Hold(ctx, channelID); err != nil {
		return fmt.Errorf("hold call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Resume(ctx context.Context, channelID string) error {
	if err := c.client.Unhold(ctx, channelID); err != nil {
		return fmt.Errorf("resume call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) PlayAudio(ctx context.Context, channelID, path string) error {
	if err := c.client.PlayAudio(ctx, channelID, path); err != nil {
		return fmt.Errorf("play audio: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) StopPlayback(ctx context.Context, channelID string) error {
	if err := c.client.StopAudio(ctx, channelID); err != nil {
		return fmt.Errorf("stop audio: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Record(
	ctx context.Context,
	channelID string,
	req RecordRequest,
) error {
	if err := c.client.Record(ctx, freeswitch.RecordRequest{
		CallID: channelID,
		Path:   req.Path,
		Action: req.Action,
	}); err != nil {
		return fmt.Errorf("record call: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) SendDTMF(ctx context.Context, channelID, digits string) error {
	if err := c.client.SendDTMF(ctx, channelID, digits); err != nil {
		return fmt.Errorf("send DTMF: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) SetCallID(
	ctx context.Context,
	channelID string,
	callID uuid.UUID,
) error {
	if callID == uuid.Nil {
		return fmt.Errorf("call id is required")
	}
	if err := c.client.SetVariable(ctx, channelID, leamoutCallIDVar, callID.String()); err != nil {
		return fmt.Errorf("set call id: %w", err)
	}
	return nil
}

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

const (
	maxToolArgumentsBytes = 64 << 10
	maxToolResponseBytes  = 64 << 10
)

type ExecuteRequest struct {
	OrganizationID uuid.UUID
	VoiceAgentID   uuid.UUID
	SessionID      uuid.UUID
	ToolID         uuid.UUID
	ToolCallID     string
	Arguments      json.RawMessage
}

type ExecuteResult struct {
	ToolID      uuid.UUID
	ToolCallID  string
	Name        string
	StatusCode  int
	ContentType string
	Body        []byte
	StartedAt   time.Time
	CompletedAt time.Time
}

type Executor struct {
	service *Service
	client  *http.Client
}

func NewExecutor(service *Service) *Executor {
	if service == nil {
		panic("tools: service is required")
	}
	return &Executor{
		service: service,
		client: &http.Client{
			Transport:     secureToolTransport(),
			CheckRedirect: rejectToolRedirect,
		},
	}
}

func (e *Executor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	if ctx == nil {
		return ExecuteResult{}, apperror.NewBadRequest("tool execution context is required")
	}
	if req.OrganizationID == uuid.Nil || req.VoiceAgentID == uuid.Nil || req.SessionID == uuid.Nil {
		return ExecuteResult{}, apperror.NewBadRequest(
			"organization_id, voice_agent_id, and session_id are required",
		)
	}
	if req.ToolID == uuid.Nil {
		return ExecuteResult{}, apperror.NewBadRequest("tool id is required")
	}
	req.ToolCallID = strings.TrimSpace(req.ToolCallID)
	if req.ToolCallID == "" || len(req.ToolCallID) > 255 {
		return ExecuteResult{}, apperror.NewBadRequest(
			"tool_call_id must be between 1 and 255 characters",
		)
	}
	if err := validateToolArguments(req.Arguments); err != nil {
		return ExecuteResult{}, err
	}

	tool, err := e.service.Get(ctx, req.OrganizationID, req.VoiceAgentID, req.ToolID)
	if err != nil {
		return ExecuteResult{}, err
	}
	if !tool.Enabled {
		return ExecuteResult{}, apperror.NewForbidden("voice agent tool is disabled")
	}
	if tool.Type != TypeWebhook {
		return ExecuteResult{}, apperror.NewNotImplemented(
			"builtin voice agent tool execution is not implemented",
		)
	}
	return e.executeWebhook(ctx, tool, req)
}

func (e *Executor) executeWebhook(
	ctx context.Context,
	tool sqlc.VoiceAgentTool,
	req ExecuteRequest,
) (ExecuteResult, error) {
	if tool.EndpointUrl == nil {
		return ExecuteResult{}, apperror.NewInternal(
			"webhook tool is missing endpoint_url",
			nil,
		)
	}
	endpoint, err := validateExecutionEndpoint(*tool.EndpointUrl)
	if err != nil {
		return ExecuteResult{}, err
	}

	body, err := json.Marshal(map[string]any{
		"id":         req.ToolCallID,
		"session_id": req.SessionID,
		"tool_id":    tool.ID,
		"tool_name":  tool.Name,
		"arguments":  json.RawMessage(req.Arguments),
	})
	if err != nil {
		return ExecuteResult{}, apperror.NewInternal("marshal tool request", err)
	}

	timeout := time.Duration(tool.TimeoutMs) * time.Millisecond
	if timeout < 100*time.Millisecond || timeout > 30*time.Second {
		return ExecuteResult{}, apperror.NewInternal("tool timeout is invalid", nil)
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(
		callCtx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return ExecuteResult{}, apperror.NewInternal("create tool request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "Leamout-Voice-Agent-Tools/1.0")
	httpReq.Header.Set("X-Leamout-Tool-Call-ID", req.ToolCallID)
	httpReq.Header.Set("X-Leamout-Session-ID", req.SessionID.String())

	startedAt := time.Now().UTC()
	resp, err := e.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			return ExecuteResult{}, apperror.NewServiceUnavailable("voice agent tool timed out", err)
		}
		return ExecuteResult{}, apperror.NewServiceUnavailable("execute voice agent tool", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := readBoundedToolResponse(resp.Body)
	if err != nil {
		return ExecuteResult{}, err
	}
	completedAt := time.Now().UTC()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ExecuteResult{}, apperror.NewServiceUnavailable(
			fmt.Sprintf("voice agent tool returned HTTP %d", resp.StatusCode),
			nil,
		)
	}
	return ExecuteResult{
		ToolID:      tool.ID,
		ToolCallID:  req.ToolCallID,
		Name:        tool.Name,
		StatusCode:  resp.StatusCode,
		ContentType: strings.TrimSpace(resp.Header.Get("Content-Type")),
		Body:        payload,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
	}, nil
}

func validateToolArguments(arguments json.RawMessage) error {
	if len(arguments) == 0 {
		return apperror.NewBadRequest("tool arguments are required")
	}
	if len(arguments) > maxToolArgumentsBytes {
		return apperror.NewPayloadTooLarge("tool arguments exceed 64 KiB")
	}
	if !json.Valid(arguments) {
		return apperror.NewBadRequest("tool arguments must be valid JSON")
	}
	var object map[string]any
	if err := json.Unmarshal(arguments, &object); err != nil || object == nil {
		return apperror.NewBadRequest("tool arguments must be a JSON object")
	}
	return nil
}

func validateExecutionEndpoint(raw string) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" {
		return nil, apperror.NewBadRequest(
			"webhook tool execution requires an absolute HTTPS endpoint",
		)
	}
	if endpoint.User != nil {
		return nil, apperror.NewBadRequest("webhook tool endpoint cannot contain userinfo")
	}
	if endpoint.Fragment != "" {
		return nil, apperror.NewBadRequest("webhook tool endpoint cannot contain a fragment")
	}
	return endpoint, nil
}

func secureToolTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("split tool endpoint address: %w", err)
		}
		addresses, err := resolveToolHost(ctx, host)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, ip := range addresses {
			conn, dialErr := dialer.DialContext(
				ctx,
				network,
				net.JoinHostPort(ip.String(), port),
			)
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("tool endpoint resolved to no dialable addresses")
		}
		return nil, fmt.Errorf("dial tool endpoint: %w", lastErr)
	}
	transport.MaxIdleConns = 20
	transport.MaxIdleConnsPerHost = 2
	transport.IdleConnTimeout = 30 * time.Second
	transport.ResponseHeaderTimeout = 10 * time.Second
	return transport
}

func resolveToolHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if parsed, err := netip.ParseAddr(host); err == nil {
		if err := validateToolAddress(parsed); err != nil {
			return nil, err
		}
		return []netip.Addr{parsed}, nil
	}

	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve tool endpoint: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("tool endpoint resolved to no addresses")
	}
	for _, address := range addresses {
		if err := validateToolAddress(address); err != nil {
			return nil, err
		}
	}
	return addresses, nil
}

func validateToolAddress(address netip.Addr) error {
	address = address.Unmap()
	if !address.IsValid() ||
		!address.IsGlobalUnicast() ||
		address.IsPrivate() ||
		address.IsLoopback() ||
		address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() ||
		address.IsMulticast() ||
		address.IsUnspecified() ||
		isCarrierGradeNAT(address) {
		return fmt.Errorf("tool endpoint resolves to a non-public address")
	}
	return nil
}

func isCarrierGradeNAT(address netip.Addr) bool {
	if !address.Is4() {
		return false
	}
	prefix := netip.MustParsePrefix("100.64.0.0/10")
	return prefix.Contains(address)
}

func rejectToolRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func readBoundedToolResponse(body io.Reader) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(body, maxToolResponseBytes+1))
	if err != nil {
		return nil, apperror.NewServiceUnavailable("read voice agent tool response", err)
	}
	if len(payload) > maxToolResponseBytes {
		return nil, apperror.NewPayloadTooLarge("voice agent tool response exceeds 64 KiB")
	}
	return payload, nil
}

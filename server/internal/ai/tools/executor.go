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

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxToolArgumentsBytes = 64 << 10
	maxToolResponseBytes  = 64 << 10
)

type ExecuteRequest struct {
	OrganizationID uuid.UUID
	VoiceAgentID   uuid.UUID
	SessionID      uuid.UUID
	CallID         uuid.UUID
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
	Replayed    bool
}

type Executor struct {
	service *Service
	calls   *calls.Service
	client  *http.Client
}

func NewExecutor(service *Service, callServices ...*calls.Service) *Executor {
	if service == nil {
		panic("tools: service is required")
	}
	var callService *calls.Service
	if len(callServices) > 0 {
		callService = callServices[0]
	}
	return &Executor{
		service: service,
		calls:   callService,
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
	if req.OrganizationID == uuid.Nil ||
		req.VoiceAgentID == uuid.Nil ||
		req.SessionID == uuid.Nil ||
		req.CallID == uuid.Nil {
		return ExecuteResult{}, apperror.NewBadRequest(
			"organization_id, voice_agent_id, session_id, and call_id are required",
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

	executionID, err := e.service.repo.ClaimExecution(ctx, req)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return ExecuteResult{}, apperror.NewInternal("claim voice agent tool execution", err)
		}
		return e.replay(ctx, tool, req)
	}

	startedAt := time.Now().UTC()
	var result ExecuteResult
	switch tool.Type {
	case TypeWebhook:
		result, err = e.executeWebhook(ctx, tool, req)
	case TypeBuiltin:
		result, err = e.executeBuiltin(ctx, tool, req)
	default:
		err = fmt.Errorf("unsupported Voice Agent tool type %q", tool.Type)
	}
	if err != nil {
		markErr := e.markFailed(ctx, req.OrganizationID, executionID, err)
		return ExecuteResult{}, errors.Join(err, markErr)
	}

	result.StartedAt = startedAt
	result.CompletedAt = time.Now().UTC()
	rows, err := e.service.repo.MarkExecutionSucceeded(
		ctx,
		req.OrganizationID,
		executionID,
		result,
	)
	if err != nil {
		return ExecuteResult{}, apperror.NewInternal(
			"persist voice agent tool execution result",
			err,
		)
	}
	if rows != 1 {
		return ExecuteResult{}, apperror.NewInternal(
			"voice agent tool execution result was not persisted",
			nil,
		)
	}
	return result, nil
}

func (e *Executor) replay(
	ctx context.Context,
	tool sqlc.VoiceAgentTool,
	req ExecuteRequest,
) (ExecuteResult, error) {
	execution, err := e.service.repo.GetExecution(
		ctx,
		req.OrganizationID,
		req.SessionID,
		req.ToolCallID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ExecuteResult{}, apperror.NewNotFound(
				"active voice agent session or enabled tool not found",
			)
		}
		return ExecuteResult{}, apperror.NewInternal(
			"read voice agent tool execution",
			err,
		)
	}
	if execution.ToolID != req.ToolID ||
		execution.VoiceAgentID != req.VoiceAgentID ||
		execution.CallID != req.CallID ||
		!sameJSON(execution.Arguments, req.Arguments) {
		return ExecuteResult{}, apperror.NewConflict(
			"tool_call_id was already used for a different execution",
		)
	}

	switch execution.State {
	case "succeeded":
		status := 0
		if execution.ResponseStatus != nil {
			status = int(*execution.ResponseStatus)
		}
		contentType := ""
		if execution.ResponseContentType != nil {
			contentType = *execution.ResponseContentType
		}
		return ExecuteResult{
			ToolID:      tool.ID,
			ToolCallID:  req.ToolCallID,
			Name:        tool.Name,
			StatusCode:  status,
			ContentType: contentType,
			Body:        append([]byte(nil), execution.ResponseBody...),
			StartedAt:   pgconv.TimestamptzToTime(execution.StartedAt),
			CompletedAt: pgconv.TimestamptzToTime(execution.CompletedAt),
			Replayed:    true,
		}, nil
	case "failed":
		message := "voice agent tool execution failed"
		if execution.ErrorMessage != nil && strings.TrimSpace(*execution.ErrorMessage) != "" {
			message = *execution.ErrorMessage
		}
		return ExecuteResult{}, apperror.NewServiceUnavailable(message, nil)
	case "processing":
		return ExecuteResult{}, apperror.NewConflict(
			"voice agent tool execution is still processing or has an uncertain outcome",
		)
	default:
		return ExecuteResult{}, apperror.NewInternal(
			"voice agent tool execution has invalid state",
			nil,
		)
	}
}

func (e *Executor) markFailed(
	ctx context.Context,
	organizationID, executionID uuid.UUID,
	cause error,
) error {
	rows, err := e.service.repo.MarkExecutionFailed(
		ctx,
		organizationID,
		executionID,
		cause.Error(),
	)
	if err != nil {
		return apperror.NewInternal("persist voice agent tool failure", err)
	}
	if rows != 1 {
		return apperror.NewInternal("voice agent tool failure was not persisted", nil)
	}
	return nil
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
	secret, err := e.service.SigningSecret(
		ctx,
		req.OrganizationID,
		req.VoiceAgentID,
		tool.ID,
	)
	if err != nil {
		return ExecuteResult{}, err
	}

	body, err := json.Marshal(map[string]any{
		"id":         req.ToolCallID,
		"session_id": req.SessionID,
		"call_id":    req.CallID,
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
	now := time.Now().UTC()
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "Leamout-Voice-Agent-Tools/1.0")
	httpReq.Header.Set("X-Leamout-Tool-Call-ID", req.ToolCallID)
	httpReq.Header.Set("X-Leamout-Session-ID", req.SessionID.String())
	httpReq.Header.Set("X-Leamout-Call-ID", req.CallID.String())
	httpReq.Header.Set("X-Leamout-Timestamp", fmt.Sprintf("%d", now.Unix()))
	httpReq.Header.Set(toolSignatureHeader, signToolRequest(secret, body, now))

	resp, err := e.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			return ExecuteResult{}, apperror.NewServiceUnavailable(
				"voice agent tool timed out",
				err,
			)
		}
		return ExecuteResult{}, apperror.NewServiceUnavailable(
			"execute voice agent tool",
			err,
		)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := readBoundedToolResponse(resp.Body)
	if err != nil {
		return ExecuteResult{}, err
	}
	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
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

func sameJSON(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil ||
		json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	leftCanonical, leftErr := json.Marshal(leftValue)
	rightCanonical, rightErr := json.Marshal(rightValue)
	return leftErr == nil &&
		rightErr == nil &&
		bytes.Equal(leftCanonical, rightCanonical)
}

func validateExecutionEndpoint(raw string) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil ||
		endpoint.Scheme != "https" ||
		endpoint.Hostname() == "" {
		return nil, apperror.NewBadRequest(
			"webhook tool execution requires an absolute HTTPS endpoint",
		)
	}
	if endpoint.User != nil {
		return nil, apperror.NewBadRequest(
			"webhook tool endpoint cannot contain userinfo",
		)
	}
	if endpoint.Fragment != "" {
		return nil, apperror.NewBadRequest(
			"webhook tool endpoint cannot contain a fragment",
		)
	}
	return endpoint, nil
}

func secureToolTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = func(
		ctx context.Context,
		network, address string,
	) (net.Conn, error) {
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
			lastErr = fmt.Errorf(
				"tool endpoint resolved to no dialable addresses",
			)
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
	payload, err := io.ReadAll(io.LimitReader(
		body,
		maxToolResponseBytes+1,
	))
	if err != nil {
		return nil, apperror.NewServiceUnavailable(
			"read voice agent tool response",
			err,
		)
	}
	if len(payload) > maxToolResponseBytes {
		return nil, apperror.NewPayloadTooLarge(
			"voice agent tool response exceeds 64 KiB",
		)
	}
	return payload, nil
}

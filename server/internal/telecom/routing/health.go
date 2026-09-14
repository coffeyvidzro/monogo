package routing

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/leamout/leamout/internal/database/sqlc"
	"golang.org/x/sync/errgroup"
)

type ProbeResult struct {
	ResponseCode int32
	Latency      time.Duration
}

type EndpointProber interface {
	Probe(context.Context, sqlc.TrunkEndpoint) (ProbeResult, error)
}

type endpointHealthStore interface {
	ListTrunkEndpointsForHealthCheck(
		context.Context,
		sqlc.ListTrunkEndpointsForHealthCheckParams,
	) ([]sqlc.TrunkEndpoint, error)
	MarkTrunkEndpointHealthy(context.Context, sqlc.MarkTrunkEndpointHealthyParams) (sqlc.TrunkEndpoint, error)
	MarkTrunkEndpointProbeFailed(context.Context, sqlc.MarkTrunkEndpointProbeFailedParams) (sqlc.TrunkEndpoint, error)
}

type EndpointHealthJob struct {
	store   endpointHealthStore
	prober  EndpointProber
	now     func() time.Time
	metrics interface {
		Probe(context.Context, uuid.UUID, uuid.UUID, bool, float64)
	}
}

func NewEndpointHealthJob(store endpointHealthStore, prober EndpointProber) (*EndpointHealthJob, error) {
	if store == nil || prober == nil {
		return nil, fmt.Errorf("endpoint health store and prober are required")
	}
	return &EndpointHealthJob{store: store, prober: prober, now: time.Now}, nil
}

func (j *EndpointHealthJob) SetMetrics(metrics interface {
	Probe(context.Context, uuid.UUID, uuid.UUID, bool, float64)
}) {
	j.metrics = metrics
}

func (j *EndpointHealthJob) Run(ctx context.Context) error {
	j.runPass(ctx)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			j.runPass(ctx)
		}
	}
}

func (j *EndpointHealthJob) runPass(ctx context.Context) {
	if err := j.Check(ctx); err != nil && ctx.Err() == nil {
		log.Printf("carrier endpoint health pass failed: %v", err)
	}
}

func (j *EndpointHealthJob) Check(ctx context.Context) error {
	checkedAt := j.now().UTC()
	endpoints, err := j.store.ListTrunkEndpointsForHealthCheck(ctx, sqlc.ListTrunkEndpointsForHealthCheckParams{
		CheckedAt: postgresTimestamp(checkedAt),
		DueBefore: postgresTimestamp(checkedAt.Add(-10 * time.Second)),
		BatchSize: 100,
	})
	if err != nil {
		return fmt.Errorf("list carrier endpoints due for health check: %w", err)
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(10)
	for _, endpoint := range endpoints {
		group.Go(func() error {
			return j.checkEndpoint(groupCtx, checkedAt, endpoint)
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}

func (j *EndpointHealthJob) checkEndpoint(ctx context.Context, checkedAt time.Time, endpoint sqlc.TrunkEndpoint) error {
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	result, probeErr := j.prober.Probe(probeCtx, endpoint)
	cancel()
	latencyMS := durationMilliseconds(result.Latency)
	var err error
	if probeErr == nil {
		code := result.ResponseCode
		_, err = j.store.MarkTrunkEndpointHealthy(ctx, sqlc.MarkTrunkEndpointHealthyParams{
			CheckedAt: postgresTimestamp(checkedAt), ResponseCode: &code, LatencyMs: &latencyMS, ID: endpoint.ID,
		})
	} else {
		message := truncateProbeError(probeErr.Error())
		_, err = j.store.MarkTrunkEndpointProbeFailed(ctx, sqlc.MarkTrunkEndpointProbeFailedParams{
			FailureThreshold: 3,
			CheckedAt:        postgresTimestamp(checkedAt),
			LatencyMs:        &latencyMS,
			LastError:        &message,
			CooldownUntil:    postgresTimestamp(checkedAt.Add(30 * time.Second)),
			ID:               endpoint.ID,
		})
	}
	if err != nil {
		return fmt.Errorf("persist carrier endpoint %s health: %w", endpoint.ID, err)
	}
	if j.metrics != nil {
		j.metrics.Probe(ctx, endpoint.TrunkID, endpoint.ID, probeErr == nil, result.Latency.Seconds())
	}
	return nil
}

func postgresTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func durationMilliseconds(value time.Duration) int32 {
	if value < 0 {
		return 0
	}
	ms := value.Milliseconds()
	if ms > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(ms)
}

func truncateProbeError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

type SIPOptionsProber struct {
	dialer net.Dialer
	now    func() time.Time
	random io.Reader
}

func NewSIPOptionsProber() *SIPOptionsProber {
	return &SIPOptionsProber{now: time.Now, random: rand.Reader}
}

func (p *SIPOptionsProber) Probe(ctx context.Context, endpoint sqlc.TrunkEndpoint) (ProbeResult, error) {
	started := p.now()
	address := net.JoinHostPort(endpoint.Host, strconv.Itoa(int(endpoint.Port)))
	var conn net.Conn
	var err error
	switch endpoint.Transport {
	case "udp", "tcp":
		conn, err = p.dialer.DialContext(ctx, endpoint.Transport, address)
	case "tls":
		tlsDialer := tls.Dialer{
			NetDialer: &p.dialer,
			Config: &tls.Config{ // #nosec G402 -- certificate verification remains enabled.
				MinVersion: tls.VersionTLS12,
				ServerName: endpoint.Host,
			},
		}
		conn, err = tlsDialer.DialContext(ctx, "tcp", address)
	default:
		return ProbeResult{}, fmt.Errorf("unsupported SIP transport %q", endpoint.Transport)
	}
	if err != nil {
		return ProbeResult{Latency: p.now().Sub(started)}, fmt.Errorf("connect SIP endpoint: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	message, err := p.optionsRequest(endpoint, conn.LocalAddr())
	if err != nil {
		return ProbeResult{Latency: p.now().Sub(started)}, err
	}
	if _, err := io.WriteString(conn, message); err != nil {
		return ProbeResult{Latency: p.now().Sub(started)}, fmt.Errorf("write SIP OPTIONS: %w", err)
	}
	line, err := bufio.NewReader(io.LimitReader(conn, 4096)).ReadString('\n')
	latency := p.now().Sub(started)
	if err != nil {
		return ProbeResult{Latency: latency}, fmt.Errorf("read SIP OPTIONS response: %w", err)
	}
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "SIP/2.0" {
		return ProbeResult{Latency: latency}, fmt.Errorf("invalid SIP OPTIONS response status %q", strings.TrimSpace(line))
	}
	code, err := strconv.Atoi(fields[1])
	if err != nil || code < 100 || code > 699 {
		return ProbeResult{Latency: latency}, fmt.Errorf("invalid SIP OPTIONS response code %q", fields[1])
	}
	return ProbeResult{ResponseCode: int32(code), Latency: latency}, nil
}

func (p *SIPOptionsProber) optionsRequest(endpoint sqlc.TrunkEndpoint, local net.Addr) (string, error) {
	random := make([]byte, 12)
	if _, err := io.ReadFull(p.random, random); err != nil {
		return "", fmt.Errorf("generate SIP OPTIONS identifiers: %w", err)
	}
	token := hex.EncodeToString(random)
	localHost, localPort, err := net.SplitHostPort(local.String())
	if err != nil {
		return "", fmt.Errorf("parse local SIP probe address: %w", err)
	}
	transport := strings.ToUpper(endpoint.Transport)
	target := net.JoinHostPort(endpoint.Host, strconv.Itoa(int(endpoint.Port)))
	const requestFormat = "OPTIONS sip:%s SIP/2.0\r\n" +
		"Via: SIP/2.0/%s %s;branch=z9hG4bK-%s;rport\r\n" +
		"From: <sip:health@leamout.invalid>;tag=%s\r\n" +
		"To: <sip:%s>\r\n" +
		"Call-ID: %s@%s\r\n" +
		"CSeq: 1 OPTIONS\r\n" +
		"Max-Forwards: 1\r\n" +
		"Content-Length: 0\r\n\r\n"
	return fmt.Sprintf(
		requestFormat,
		target, transport, net.JoinHostPort(localHost, localPort), token, token, target, token, localHost,
	), nil
}

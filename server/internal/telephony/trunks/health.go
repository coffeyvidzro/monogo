package trunks

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type HealthCheckConfig struct {
	Interval         time.Duration
	EndpointTimeout  time.Duration
	FailureThreshold int32
	Cooldown         time.Duration
	BatchSize        int32
}

func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{Interval: 30 * time.Second, EndpointTimeout: 3 * time.Second, FailureThreshold: 3, Cooldown: 2 * time.Minute, BatchSize: 100}
}

type healthRepository interface {
	ListForHealthCheck(context.Context, time.Time, time.Time, int32) ([]sqlc.TrunkEndpoint, error)
	MarkHealthy(context.Context, uuid.UUID, time.Time, int32, int32) error
	MarkProbeFailed(context.Context, uuid.UUID, time.Time, int32, string, int32, time.Time) error
}

type EndpointProber interface {
	Probe(context.Context, sqlc.TrunkEndpoint) (int32, error)
}

type HealthCheckJob struct {
	repo   healthRepository
	prober EndpointProber
	config HealthCheckConfig
	now    func() time.Time
}

func NewHealthCheckJob(repo healthRepository, prober EndpointProber, config HealthCheckConfig) (*HealthCheckJob, error) {
	if repo == nil || prober == nil {
		return nil, fmt.Errorf("trunk health check requires repository and prober")
	}
	defaults := DefaultHealthCheckConfig()
	if config.Interval <= 0 {
		config.Interval = defaults.Interval
	}
	if config.EndpointTimeout <= 0 {
		config.EndpointTimeout = defaults.EndpointTimeout
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = defaults.FailureThreshold
	}
	if config.Cooldown <= 0 {
		config.Cooldown = defaults.Cooldown
	}
	if config.BatchSize <= 0 {
		config.BatchSize = defaults.BatchSize
	}
	return &HealthCheckJob{repo: repo, prober: prober, config: config, now: time.Now}, nil
}

func (j *HealthCheckJob) Run(ctx context.Context) error {
	j.runPass(ctx)
	ticker := time.NewTicker(j.config.Interval)
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

func (j *HealthCheckJob) runPass(ctx context.Context) {
	if err := j.Check(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("trunk endpoint health-check pass failed: %v", err)
	}
}

func (j *HealthCheckJob) Check(ctx context.Context) error {
	now := j.now().UTC()
	endpoints, err := j.repo.ListForHealthCheck(ctx, now, now.Add(-j.config.Interval), j.config.BatchSize)
	if err != nil {
		return fmt.Errorf("claim trunk endpoints: %w", err)
	}
	for _, endpoint := range endpoints {
		started := j.now()
		probeCtx, cancel := context.WithTimeout(ctx, j.config.EndpointTimeout)
		code, probeErr := j.prober.Probe(probeCtx, endpoint)
		cancel()
		finished := j.now().UTC()
		latency := int32(max(finished.Sub(started).Milliseconds(), 0))
		if probeErr == nil {
			if err := j.repo.MarkHealthy(ctx, endpoint.ID, finished, code, latency); err != nil {
				return fmt.Errorf("mark trunk endpoint healthy: %w", err)
			}
			continue
		}
		message := probeErr.Error()
		if len(message) > 500 {
			message = message[:500]
		}
		if err := j.repo.MarkProbeFailed(ctx, endpoint.ID, finished, latency, message, j.config.FailureThreshold, finished.Add(j.config.Cooldown)); err != nil {
			return fmt.Errorf("mark trunk endpoint failed: %w", err)
		}
	}
	return nil
}

// SIPOptionsProber sends an unauthenticated SIP OPTIONS request. Any syntactically
// valid SIP response proves transport reachability, including 401 and 407.
type SIPOptionsProber struct {
	Dialer    *net.Dialer
	TLSConfig *tls.Config
}

func (p SIPOptionsProber) Probe(ctx context.Context, endpoint sqlc.TrunkEndpoint) (int32, error) {
	address := net.JoinHostPort(endpoint.Host, strconv.Itoa(int(endpoint.Port)))
	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	payload, err := optionsRequest(endpoint.Host, endpoint.Port, endpoint.Transport)
	if err != nil {
		return 0, err
	}
	switch endpoint.Transport {
	case "udp":
		return p.probeUDP(ctx, dialer, address, payload)
	case "tls":
		cfg := p.TLSConfig
		if cfg == nil {
			cfg = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		clone := cfg.Clone()
		if clone.ServerName == "" && net.ParseIP(endpoint.Host) == nil {
			clone.ServerName = endpoint.Host
		}
		raw, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return 0, fmt.Errorf("dial SIP TLS endpoint: %w", err)
		}
		return probeStream(ctx, tls.Client(raw, clone), payload)
	case "tcp":
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return 0, fmt.Errorf("dial SIP endpoint: %w", err)
		}
		return probeStream(ctx, conn, payload)
	default:
		return 0, fmt.Errorf("unsupported SIP transport %q", endpoint.Transport)
	}
}

func probeStream(ctx context.Context, conn net.Conn, payload []byte) (int32, error) {
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if _, err := conn.Write(payload); err != nil {
		return 0, fmt.Errorf("write SIP OPTIONS: %w", err)
	}
	return readSIPStatus(bufio.NewReader(conn))
}

func (p SIPOptionsProber) probeUDP(ctx context.Context, dialer *net.Dialer, address string, payload []byte) (int32, error) {
	conn, err := dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return 0, fmt.Errorf("dial SIP UDP endpoint: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if _, err := conn.Write(payload); err != nil {
		return 0, fmt.Errorf("write SIP OPTIONS: %w", err)
	}
	buffer := make([]byte, 64<<10)
	n, err := conn.Read(buffer)
	if err != nil {
		return 0, fmt.Errorf("read SIP response: %w", err)
	}
	return parseSIPStatus(string(buffer[:n]))
}

func optionsRequest(host string, port int32, transport string) ([]byte, error) {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate SIP probe id: %w", err)
	}
	id := hex.EncodeToString(nonce)
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))
	message := fmt.Sprintf("OPTIONS sip:%s SIP/2.0\r\nVia: SIP/2.0/%s 127.0.0.1:5060;branch=z9hG4bK-%s;rport\r\nMax-Forwards: 1\r\nFrom: <sip:health@localhost>;tag=%s\r\nTo: <sip:%s>\r\nCall-ID: %s@localhost\r\nCSeq: 1 OPTIONS\r\nContact: <sip:health@127.0.0.1:5060>\r\nContent-Length: 0\r\n\r\n", target, strings.ToUpper(transport), id, id, target, id)
	return []byte(message), nil
}

func readSIPStatus(reader *bufio.Reader) (int32, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("read SIP response: %w", err)
	}
	return parseSIPStatus(line)
}

func parseSIPStatus(value string) (int32, error) {
	line := strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
	parts := strings.Fields(line)
	if len(parts) < 2 || parts[0] != "SIP/2.0" {
		return 0, errors.New("endpoint returned an invalid SIP response")
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil || code < 100 || code > 699 {
		return 0, errors.New("endpoint returned an invalid SIP status")
	}
	return int32(code), nil
}

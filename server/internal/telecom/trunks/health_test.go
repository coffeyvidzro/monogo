package trunks

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

func TestSIPOptionsProberAcceptsAuthenticationChallenge(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	listener, err := (&net.ListenConfig{}).ListenPacket(ctx, "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close UDP listener: %v", err)
		}
	})
	done := make(chan error, 1)
	go func() {
		buffer := make([]byte, 4096)
		n, peer, err := listener.ReadFrom(buffer)
		if err != nil {
			done <- err
			return
		}
		if !strings.HasPrefix(string(buffer[:n]), "OPTIONS sip:") {
			done <- context.Canceled
			return
		}
		_, err = listener.WriteTo([]byte("SIP/2.0 401 Unauthorized\r\nContent-Length: 0\r\n\r\n"), peer)
		done <- err
	}()
	address := listener.LocalAddr().(*net.UDPAddr)
	code, err := (SIPOptionsProber{}).Probe(ctx, sqlc.TrunkEndpoint{Host: "127.0.0.1", Port: int32(address.Port), Transport: "udp"})
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if code != 401 {
		t.Fatalf("Probe() code = %d, want 401", code)
	}
	if err := <-done; err != nil {
		t.Fatalf("server error = %v", err)
	}
}

type fakeHealthRepository struct {
	endpoints       []sqlc.TrunkEndpoint
	healthy, failed []uuid.UUID
}

func (f *fakeHealthRepository) ListForHealthCheck(context.Context, time.Time, time.Time, int32) ([]sqlc.TrunkEndpoint, error) {
	return f.endpoints, nil
}
func (f *fakeHealthRepository) MarkHealthy(_ context.Context, id uuid.UUID, _ time.Time, _, _ int32) error {
	f.healthy = append(f.healthy, id)
	return nil
}
func (f *fakeHealthRepository) MarkProbeFailed(_ context.Context, id uuid.UUID, _ time.Time, _ int32, _ string, _ int32, _ time.Time) error {
	f.failed = append(f.failed, id)
	return nil
}

type fakeProber struct {
	mu       sync.Mutex
	failures map[uuid.UUID]bool
}

func (f *fakeProber) Probe(_ context.Context, endpoint sqlc.TrunkEndpoint) (int32, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failures[endpoint.ID] {
		return 0, context.DeadlineExceeded
	}
	return 200, nil
}

func TestHealthCheckJobPersistsProbeResults(t *testing.T) {
	healthyID, failedID := uuid.New(), uuid.New()
	repo := &fakeHealthRepository{endpoints: []sqlc.TrunkEndpoint{{ID: healthyID}, {ID: failedID}}}
	job, err := NewHealthCheckJob(repo, &fakeProber{failures: map[uuid.UUID]bool{failedID: true}}, DefaultHealthCheckConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(repo.healthy) != 1 || repo.healthy[0] != healthyID {
		t.Fatalf("healthy = %v", repo.healthy)
	}
	if len(repo.failed) != 1 || repo.failed[0] != failedID {
		t.Fatalf("failed = %v", repo.failed)
	}
}

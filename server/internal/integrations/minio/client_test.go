package minio

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig(" leamout.example ", "app-key", "app-secret")
	if cfg.Endpoint != "http://minio:9000" || cfg.PublicEndpoint != "https://recordings.leamout.example" {
		t.Fatalf("unexpected MinIO endpoints: internal=%q public=%q", cfg.Endpoint, cfg.PublicEndpoint)
	}
	if cfg.Region != "us-east-1" || cfg.Bucket != "recordings" || !cfg.UsePathStyle || cfg.PlaybackTTL != 15*time.Minute {
		t.Fatalf("unexpected MinIO defaults: %+v", cfg)
	}
	if cfg.AccessKey != "app-key" || cfg.SecretKey != "app-secret" {
		t.Fatal("application credentials were not preserved")
	}
}

package config

import (
	"testing"

	"github.com/caarlos0/env/v11"
)

func TestMinIOAppCredentialsFromEnvironment(t *testing.T) {
	t.Setenv("MINIO_APP_ACCESS_KEY", "app-access")
	t.Setenv("MINIO_APP_SECRET_KEY", "app-secret")

	cfg, err := env.ParseAs[struct {
		MinIO MinIOConfig `envPrefix:"MINIO_"`
	}]()
	if err != nil {
		t.Fatalf("parse MinIO credentials: %v", err)
	}
	if cfg.MinIO.AccessKey != "app-access" || cfg.MinIO.SecretKey != "app-secret" {
		t.Fatalf("MinIO credentials not loaded correctly")
	}
}

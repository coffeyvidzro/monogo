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

func TestRequiredProviderCredentials(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
	}{
		{name: "missing DIDWW", edit: func(c *Config) { c.DIDWW.APIKey = "   " }},
		{name: "missing CommPeak", edit: func(c *Config) { c.CommPeak.Authorization = "   " }},
		{name: "missing Stripe", edit: func(c *Config) { c.Stripe.SecretKey = "   " }},
		{name: "missing Stripe webhook secret", edit: func(c *Config) { c.Stripe.WebhookSecret = "   " }},
		{name: "missing Paystack", edit: func(c *Config) { c.Paystack.SecretKey = "   " }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				DIDWW: DIDWWConfig{APIKey: "didww-test"},
				CommPeak: CommPeakConfig{Authorization: "commpeak-test"},
				Stripe: StripeConfig{SecretKey: "stripe-test", WebhookSecret: "webhook-test"},
				Paystack: PaystackConfig{SecretKey: "paystack-test"},
			}
			tc.edit(&cfg)
			cfg.normalize()
			if err := cfg.validateCredentials(); err == nil {
				t.Fatal("expected missing provider credential to be rejected")
			}
		})
	}
}

func TestRequiredProviderEnvironmentTags(t *testing.T) {
	for _, name := range []string{
		"DIDWW_API_KEY",
		"COMMPEAK_API_AUTHORIZATION",
		"STRIPE_SECRET_KEY",
		"STRIPE_WEBHOOK_SECRET",
		"PAYSTACK_SECRET_KEY",
	} {
		t.Setenv(name, "")
	}

	_, err := env.ParseAs[struct {
		DIDWW DIDWWConfig `envPrefix:"DIDWW_"`
		CommPeak CommPeakConfig `envPrefix:"COMMPEAK_"`
		Stripe StripeConfig `envPrefix:"STRIPE_"`
		Paystack PaystackConfig `envPrefix:"PAYSTACK_"`
	}]()
	if err == nil {
		t.Fatal("required provider environment variables must not be empty")
	}
}

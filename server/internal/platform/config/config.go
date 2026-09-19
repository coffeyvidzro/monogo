package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type DIDWWConfig struct {
	APIKey     string `env:"API_KEY"`
	APIBaseURL string `env:"API_BASE_URL" envDefault:"https://api.didww.com/v3"`
}

type CommPeakConfig struct {
	Authorization string `env:"API_AUTHORIZATION"`
	APIBaseURL    string `env:"API_BASE_URL" envDefault:"https://api.commpeak.com"`
}

type StripeConfig struct {
	SecretKey     string `env:"SECRET_KEY"`
	WebhookSecret string `env:"WEBHOOK_SECRET"`
	APIBaseURL    string `env:"API_BASE_URL" envDefault:"https://api.stripe.com/v1"`
}

type PaystackConfig struct {
	SecretKey  string `env:"SECRET_KEY"`
	APIBaseURL string `env:"API_BASE_URL" envDefault:"https://api.paystack.co"`
}

type S3Config struct {
	Endpoint       string        `env:"ENDPOINT,required"`
	PublicEndpoint string        `env:"PUBLIC_ENDPOINT,required"`
	Region         string        `env:"REGION" envDefault:"us-east-1"`
	Bucket         string        `env:"BUCKET,required"`
	AccessKey      string        `env:"ACCESS_KEY,required"`
	SecretKey      string        `env:"SECRET_KEY,required"`
	UsePathStyle   bool          `env:"USE_PATH_STYLE" envDefault:"true"`
	PlaybackTTL    time.Duration `env:"PLAYBACK_TTL" envDefault:"15m"`
	StagingPath    string        `env:"STAGING_PATH" envDefault:"/var/lib/freeswitch/recordings"`
}

type Config struct {
	AppEnv                string         `env:"APP_ENV" envDefault:"development"`
	Domain                string         `env:"DOMAIN"`
	DatabaseURL           string         `env:"DATABASE_URL,required"`
	RedisURL              string         `env:"REDIS_URL,required"`
	NATSURL               string         `env:"NATS_URL,required"`
	FreeSWITCHESLAddress  string         `env:"FREESWITCH_ESL_ADDRESS" envDefault:"127.0.0.1:8021"`
	FreeSWITCHESLPassword string         `env:"FREESWITCH_ESL_PASSWORD,required"`
	EncryptionKey         string         `env:"ENCRYPTION_KEY,required"`
	DIDWW                 DIDWWConfig    `envPrefix:"DIDWW_"`
	CommPeak              CommPeakConfig `envPrefix:"COMMPEAK_"`
	Stripe                StripeConfig   `envPrefix:"STRIPE_"`
	Paystack              PaystackConfig `envPrefix:"PAYSTACK_"`
	S3                    S3Config       `envPrefix:"S3_"`
	TURNAuthSecret        string         `env:"TURN_AUTH_SECRET,required"`
	TURNPublicURLs        []string       `env:"TURN_PUBLIC_URLS" envSeparator:"," envDefault:"stun:localhost:3478,turn:localhost:3478?transport=udp,turn:localhost:3478?transport=tcp"`
	CORSOrigins           []string       `env:"CORS_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000,http://127.0.0.1:3000"`
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}

	cfg.normalize()

	return cfg, nil
}

func (c Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development")
}

func (c *Config) normalize() {
	c.AppEnv = strings.TrimSpace(c.AppEnv)
	c.Domain = strings.TrimSpace(c.Domain)
	c.DatabaseURL = strings.TrimSpace(c.DatabaseURL)
	c.RedisURL = strings.TrimSpace(c.RedisURL)
	c.NATSURL = strings.TrimSpace(c.NATSURL)
	c.FreeSWITCHESLAddress = strings.TrimSpace(c.FreeSWITCHESLAddress)
	c.FreeSWITCHESLPassword = strings.TrimSpace(c.FreeSWITCHESLPassword)
	c.EncryptionKey = strings.TrimSpace(c.EncryptionKey)
	c.DIDWW.APIKey = strings.TrimSpace(c.DIDWW.APIKey)
	c.DIDWW.APIBaseURL = strings.TrimRight(strings.TrimSpace(c.DIDWW.APIBaseURL), "/")
	c.CommPeak.Authorization = strings.TrimSpace(c.CommPeak.Authorization)
	c.CommPeak.APIBaseURL = strings.TrimRight(strings.TrimSpace(c.CommPeak.APIBaseURL), "/")
	c.Stripe.SecretKey = strings.TrimSpace(c.Stripe.SecretKey)
	c.Stripe.WebhookSecret = strings.TrimSpace(c.Stripe.WebhookSecret)
	c.Stripe.APIBaseURL = strings.TrimRight(strings.TrimSpace(c.Stripe.APIBaseURL), "/")
	c.Paystack.SecretKey = strings.TrimSpace(c.Paystack.SecretKey)
	c.Paystack.APIBaseURL = strings.TrimRight(strings.TrimSpace(c.Paystack.APIBaseURL), "/")
	c.S3.Endpoint = strings.TrimRight(strings.TrimSpace(c.S3.Endpoint), "/")
	c.S3.PublicEndpoint = strings.TrimRight(strings.TrimSpace(c.S3.PublicEndpoint), "/")
	c.S3.Region = strings.TrimSpace(c.S3.Region)
	c.S3.Bucket = strings.TrimSpace(c.S3.Bucket)
	c.S3.AccessKey = strings.TrimSpace(c.S3.AccessKey)
	c.S3.SecretKey = strings.TrimSpace(c.S3.SecretKey)
	c.S3.StagingPath = strings.TrimRight(strings.TrimSpace(c.S3.StagingPath), "/")
	c.TURNAuthSecret = strings.TrimSpace(c.TURNAuthSecret)
	c.TURNPublicURLs = normalizeStrings(c.TURNPublicURLs)
	c.CORSOrigins = normalizeStrings(c.CORSOrigins)
}

func normalizeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

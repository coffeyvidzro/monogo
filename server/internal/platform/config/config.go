package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type DIDWWConfig struct {
	APIKey     string `env:"API_KEY,required"`
	APIBaseURL string `env:"API_BASE_URL" envDefault:"https://api.didww.com/v3"`
}

type CommPeakConfig struct {
	Authorization string `env:"API_AUTHORIZATION,required"`
	APIBaseURL    string `env:"API_BASE_URL" envDefault:"https://api.commpeak.com"`
}

type MinIOConfig struct {
	AccessKey string `env:"APP_ACCESS_KEY,required"`
	SecretKey string `env:"APP_SECRET_KEY,required"`
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
	MinIO                 MinIOConfig    `envPrefix:"MINIO_"`
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
	c.MinIO.AccessKey = strings.TrimSpace(c.MinIO.AccessKey)
	c.MinIO.SecretKey = strings.TrimSpace(c.MinIO.SecretKey)
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

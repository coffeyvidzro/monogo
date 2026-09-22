package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint       string
	PublicEndpoint string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	UsePathStyle   bool
	PlaybackTTL    time.Duration
}

// DefaultConfig selects the fixed settings for the self-hosted MinIO deployment.
// Only the application credentials and deployment domain come from the environment.
func DefaultConfig(domain, accessKey, secretKey string) Config {
	return Config{
		Endpoint:       "http://minio:9000",
		PublicEndpoint: "https://recordings." + strings.TrimSpace(domain),
		Region:         "us-east-1",
		Bucket:         "recordings",
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		UsePathStyle:   true,
		PlaybackTTL:    15 * time.Minute,
	}
}

type Client struct {
	client      *miniosdk.Client
	presigner   *miniosdk.Client
	bucket      string
	playbackTTL time.Duration
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	endpoint, secure, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("S3 credentials are required")
	}
	if cfg.PlaybackTTL <= 0 {
		cfg.PlaybackTTL = 15 * time.Minute
	}
	client, err := miniosdk.New(endpoint, &miniosdk.Options{
		Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: secure, Region: strings.TrimSpace(cfg.Region), BucketLookup: bucketLookup(cfg.UsePathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("initialize S3 client: %w", err)
	}
	publicEndpoint, publicSecure, err := parseEndpoint(cfg.PublicEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid public S3 endpoint: %w", err)
	}
	presigner, err := miniosdk.New(publicEndpoint, &miniosdk.Options{
		Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: publicSecure,
		Region: strings.TrimSpace(cfg.Region), BucketLookup: bucketLookup(cfg.UsePathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("initialize public S3 signer: %w", err)
	}
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check S3 bucket: %w", err)
	}
	if !exists {
		// Acceptance tests use disposable MinIO administrator credentials.
		// Production requires a pre-provisioned bucket and a scoped user.
		if err := client.MakeBucket(ctx, cfg.Bucket, miniosdk.MakeBucketOptions{Region: cfg.Region}); err != nil {
			// The API and worker can race to create the same bucket.
			ready, checkErr := client.BucketExists(ctx, cfg.Bucket)
			if checkErr != nil || !ready {
				return nil, fmt.Errorf("create S3 bucket %q: %w", cfg.Bucket, err)
			}
		}
	}
	return &Client{client: client, presigner: presigner, bucket: cfg.Bucket, playbackTTL: cfg.PlaybackTTL}, nil
}

func (c *Client) Put(ctx context.Context, key, contentType string, reader io.Reader, size int64) error {
	if err := validateKey(key); err != nil {
		return err
	}
	_, err := c.client.PutObject(ctx, c.bucket, key, reader, size, miniosdk.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("upload S3 object: %w", err)
	}
	return nil
}

func (c *Client) PlaybackURL(ctx context.Context, key string) (string, time.Time, error) {
	if err := validateKey(key); err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(c.playbackTTL)
	value, err := c.presigner.PresignedGetObject(ctx, c.bucket, key, c.playbackTTL, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presign S3 object: %w", err)
	}
	return value.String(), expiresAt, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := c.client.RemoveObject(ctx, c.bucket, key, miniosdk.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}

func (c *Client) Bucket() string { return c.bucket }

func parseEndpoint(value string) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", false, fmt.Errorf("S3 endpoint must be an HTTP or HTTPS origin")
	}
	return parsed.Host, parsed.Scheme == "https", nil
}

func bucketLookup(pathStyle bool) miniosdk.BucketLookupType {
	if pathStyle {
		return miniosdk.BucketLookupPath
	}
	return miniosdk.BucketLookupAuto
}

func validateKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return fmt.Errorf("S3 object key is invalid")
	}
	return nil
}

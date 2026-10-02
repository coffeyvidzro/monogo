package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
	PlaybackTTL  time.Duration
}

type Client struct {
	client      *miniosdk.Client
	bucket      string
	playbackTTL time.Duration
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	endpoint, secure, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.AccessKey = strings.TrimSpace(cfg.AccessKey)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("S3 credentials are required")
	}
	if cfg.PlaybackTTL <= 0 {
		cfg.PlaybackTTL = 15 * time.Minute
	}

	client, err := miniosdk.New(endpoint, &miniosdk.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       secure,
		Region:       cfg.Region,
		BucketLookup: bucketLookup(cfg.UsePathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("initialize S3 client: %w", err)
	}

	return &Client{
		client:      client,
		bucket:      cfg.Bucket,
		playbackTTL: cfg.PlaybackTTL,
	}, nil
}

func (c *Client) Test(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check S3 bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("S3 bucket %q does not exist", c.bucket)
	}

	key := "leamout-storage-test/" + uuid.NewString()
	body := "leamout"
	if _, err := c.client.PutObject(
		ctx,
		c.bucket,
		key,
		strings.NewReader(body),
		int64(len(body)),
		miniosdk.PutObjectOptions{ContentType: "text/plain"},
	); err != nil {
		return fmt.Errorf("test S3 write access: %w", err)
	}
	if err := c.client.RemoveObject(
		ctx,
		c.bucket,
		key,
		miniosdk.RemoveObjectOptions{},
	); err != nil {
		return fmt.Errorf("test S3 delete access: %w", err)
	}
	return nil
}

func (c *Client) Put(
	ctx context.Context,
	key string,
	contentType string,
	reader io.Reader,
	size int64,
) error {
	if err := validateKey(key); err != nil {
		return err
	}
	_, err := c.client.PutObject(
		ctx,
		c.bucket,
		key,
		reader,
		size,
		miniosdk.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return fmt.Errorf("upload S3 object: %w", err)
	}
	return nil
}

func (c *Client) PlaybackURL(
	ctx context.Context,
	key string,
) (string, time.Time, error) {
	if err := validateKey(key); err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(c.playbackTTL)
	value, err := c.client.PresignedGetObject(
		ctx,
		c.bucket,
		key,
		c.playbackTTL,
		nil,
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presign S3 object: %w", err)
	}
	return value.String(), expiresAt, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := c.client.RemoveObject(
		ctx,
		c.bucket,
		key,
		miniosdk.RemoveObjectOptions{},
	); err != nil {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}

func (c *Client) Bucket() string { return c.bucket }

func parseEndpoint(value string) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
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

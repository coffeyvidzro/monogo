package storage

import (
	"time"

	"github.com/google/uuid"
)

const (
	ProviderS3        = "s3"
	PurposeRecordings = "recordings"
	StatusActive      = "active"
	StatusDisabled    = "disabled"
)

type Integration struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Provider       string
	Purpose        string
	EndpointURL    string
	Region         string
	Bucket         string
	AccessKeyID    string
	UsePathStyle   bool
	Status         string
	HasSecret      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateRequest struct {
	Name            string `json:"name"`
	EndpointURL     string `json:"endpoint_url"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UsePathStyle    bool   `json:"use_path_style"`
}

// UpdateRequest intentionally excludes endpoint, region, bucket, and path-style
// settings. A storage integration is the durable identity of the object store
// that owns historical recordings. To move recordings to a new destination,
// disable this integration and create another one instead.
type UpdateRequest struct {
	Name            *string `json:"name,omitempty"`
	AccessKeyID     *string `json:"access_key_id,omitempty"`
	SecretAccessKey *string `json:"secret_access_key,omitempty"`
	Status          *string `json:"status,omitempty"`
}

type Response struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Provider       string    `json:"provider"`
	Purpose        string    `json:"purpose"`
	EndpointURL    string    `json:"endpoint_url"`
	Region         string    `json:"region"`
	Bucket         string    `json:"bucket"`
	AccessKeyID    string    `json:"access_key_id"`
	UsePathStyle   bool      `json:"use_path_style"`
	Status         string    `json:"status"`
	HasSecret      bool      `json:"has_secret"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ResolvedIntegration struct {
	Integration
	SecretAccessKey string
}

func response(value Integration) Response {
	return Response(value)
}

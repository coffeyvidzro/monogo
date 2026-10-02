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

type UpdateRequest struct {
	Name            *string `json:"name,omitempty"`
	EndpointURL     *string `json:"endpoint_url,omitempty"`
	Region          *string `json:"region,omitempty"`
	Bucket          *string `json:"bucket,omitempty"`
	AccessKeyID     *string `json:"access_key_id,omitempty"`
	SecretAccessKey *string `json:"secret_access_key,omitempty"`
	UsePathStyle    *bool   `json:"use_path_style,omitempty"`
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
	return Response{
		ID:             value.ID,
		OrganizationID: value.OrganizationID,
		Name:           value.Name,
		Provider:       value.Provider,
		Purpose:        value.Purpose,
		EndpointURL:    value.EndpointURL,
		Region:         value.Region,
		Bucket:         value.Bucket,
		AccessKeyID:    value.AccessKeyID,
		UsePathStyle:   value.UsePathStyle,
		Status:         value.Status,
		HasSecret:      value.HasSecret,
		CreatedAt:      value.CreatedAt,
		UpdatedAt:      value.UpdatedAt,
	}
}

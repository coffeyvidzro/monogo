package numbers

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type EmergencyAddressRequest struct {
	Name         string `json:"name"`
	AddressLine1 string `json:"address_line1"`
	AddressLine2 string `json:"address_line2,omitempty"`
	Locality     string `json:"locality"`
	Region       string `json:"region"`
	PostalCode   string `json:"postal_code"`
	CountryCode  string `json:"country_code"`
}

type PortInRequest struct {
	Number          string                  `json:"number"`
	LosingCarrier   string                  `json:"losing_carrier"`
	AccountNumber   string                  `json:"account_number"`
	AuthorizedName  string                  `json:"authorized_name"`
	ServiceAddress  EmergencyAddressRequest `json:"service_address"`
	DesiredPortDate *time.Time              `json:"desired_port_date,omitempty"`
}

type PortDocumentRequest struct {
	DocumentType string `json:"document_type"`
	ObjectKey    string `json:"object_key"`
	SHA256       string `json:"sha256"`
}

type PortInResponse struct {
	Case      sqlc.PortInCase
	Operation sqlc.NumberLifecycleOperation
	Documents []sqlc.PortInDocument
}

func lifecycleResponse(operation sqlc.NumberLifecycleOperation) map[string]any {
	return map[string]any{
		"id": operation.ID, "phone_number_id": operation.PhoneNumberID,
		"operation": operation.Operation, "status": operation.Status,
		"number": operation.RequestedNumber, "submitted_at": operation.SubmittedAt,
		"completed_at": operation.CompletedAt, "created_at": operation.CreatedAt,
		"updated_at": operation.UpdatedAt,
	}
}

func emergencyResponse(registration sqlc.EmergencyRegistration) map[string]any {
	return map[string]any{
		"id": registration.ID, "phone_number_id": registration.PhoneNumberID,
		"status": registration.Status, "name": registration.Name,
		"address_line1": registration.AddressLine1, "address_line2": registration.AddressLine2,
		"locality": registration.Locality, "region": registration.Region,
		"postal_code": registration.PostalCode, "country_code": registration.CountryCode,
		"validation_message": registration.ValidationMessage,
		"activated_at":       registration.ActivatedAt, "created_at": registration.CreatedAt,
		"updated_at": registration.UpdatedAt,
	}
}

func portInResponse(response PortInResponse) map[string]any {
	documents := make([]map[string]any, 0, len(response.Documents))
	for _, document := range response.Documents {
		documents = append(documents, portDocumentResponse(document))
	}
	return map[string]any{
		"id": response.Case.ID, "number": response.Operation.RequestedNumber,
		"status": response.Case.Status, "losing_carrier": response.Case.LosingCarrier,
		"authorized_name":   response.Case.AuthorizedName,
		"desired_port_date": response.Case.DesiredPortDate, "foc_at": response.Case.FocAt,
		"activated_at": response.Case.ActivatedAt, "rejection_code": response.Case.RejectionCode,
		"rejection_message": response.Case.RejectionMessage, "documents": documents,
		"created_at": response.Case.CreatedAt, "updated_at": response.Case.UpdatedAt,
	}
}

func portDocumentResponse(document sqlc.PortInDocument) map[string]any {
	return map[string]any{
		"id": document.ID, "document_type": document.DocumentType,
		"object_key": document.ObjectKey, "sha256": document.Sha256,
		"status": document.Status, "created_at": document.CreatedAt,
	}
}

type NumberLifecycleProvider interface {
	RequestRelease(ctx context.Context, providerResourceID string) (string, error)
	ReleaseCompleted(ctx context.Context, providerResourceID string) (bool, error)
	ValidateEmergency(ctx context.Context, number string, address EmergencyAddressRequest) (string, bool, string, error)
	CheckPortability(ctx context.Context, number string) (bool, string, error)
	SubmitPortIn(ctx context.Context, operationID uuid.UUID, request PortInRequest, documents []sqlc.PortInDocument) (string, error)
	PortInStatus(ctx context.Context, providerReference string) (ProviderPortStatus, error)
}

type ProviderPortStatus struct {
	Status             string
	FOCAt              *time.Time
	ProviderResourceID string
	RejectionCode      string
	RejectionMessage   string
}

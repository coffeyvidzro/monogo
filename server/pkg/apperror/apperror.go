package apperror

import (
	"fmt"
	"net/http"
)

// AppError is a structured application error.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Err     error  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf(
			"%s: %s: %v",
			e.Code,
			e.Message,
			e.Err,
		)
	}

	return fmt.Sprintf(
		"%s: %s",
		e.Code,
		e.Message,
	)
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewNotFound creates a not found error.
func NewNotFound(message string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: message,
		Status:  http.StatusNotFound,
	}
}

// NewBadRequest creates a bad request error.
func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:    "BAD_REQUEST",
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

// NewPaymentRequired creates a payment required error.
func NewPaymentRequired(message string) *AppError {
	return &AppError{
		Code:    "PAYMENT_REQUIRED",
		Message: message,
		Status:  http.StatusPaymentRequired,
	}
}

// NewPayloadTooLarge creates a payload too large error.
func NewPayloadTooLarge(message string) *AppError {
	return &AppError{
		Code:    "PAYLOAD_TOO_LARGE",
		Message: message,
		Status:  http.StatusRequestEntityTooLarge,
	}
}

// NewUnauthorized creates an unauthorized error.
func NewUnauthorized(message string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

// NewForbidden creates a forbidden error.
func NewForbidden(message string) *AppError {
	return &AppError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  http.StatusForbidden,
	}
}

// NewStepUpRequired creates a step-up authentication required error.
func NewStepUpRequired(message string) *AppError {
	return &AppError{
		Code:    "STEP_UP_REQUIRED",
		Message: message,
		Status:  http.StatusForbidden,
	}
}

// NewConflict creates a conflict error.
func NewConflict(message string) *AppError {
	return &AppError{
		Code:    "CONFLICT",
		Message: message,
		Status:  http.StatusConflict,
	}
}

// NewTooManyRequests creates a rate limit error.
func NewTooManyRequests(message string) *AppError {
	return &AppError{
		Code:    "TOO_MANY_REQUESTS",
		Message: message,
		Status:  http.StatusTooManyRequests,
	}
}

// NewNotImplemented creates a not implemented error.
func NewNotImplemented(message string) *AppError {
	return &AppError{
		Code:    "NOT_IMPLEMENTED",
		Message: message,
		Status:  http.StatusNotImplemented,
	}
}

// NewInternal creates an internal server error.
func NewInternal(
	message string,
	err error,
) *AppError {
	return &AppError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

// NewServiceUnavailable creates a service unavailable error.
func NewServiceUnavailable(
	message string,
	err error,
) *AppError {
	return &AppError{
		Code:    "SERVICE_UNAVAILABLE",
		Message: message,
		Status:  http.StatusServiceUnavailable,
		Err:     err,
	}
}

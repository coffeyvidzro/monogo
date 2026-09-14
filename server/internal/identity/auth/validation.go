package auth

import (
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeOTP(code string) string {
	return strings.TrimSpace(code)
}

func validateStartRequest(req startRequest) (string, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return "", apperror.NewBadRequest("email is required")
	}

	return email, nil
}

func validateTransactionID(value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, apperror.NewBadRequest("transaction_id is required")
	}

	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.NewBadRequest("invalid transaction_id")
	}

	return id, nil
}

func validatePasswordLoginRequest(req passwordLoginRequest) (uuid.UUID, error) {
	transactionID, err := validateTransactionID(req.TransactionID)
	if err != nil {
		return uuid.Nil, err
	}

	if req.Password == "" {
		return uuid.Nil, apperror.NewBadRequest("password is required")
	}

	return transactionID, nil
}

func validateSendOTPRequest(req sendOTPRequest) (uuid.UUID, error) {
	return validateTransactionID(req.TransactionID)
}

func validateVerifyOTPRequest(req verifyOTPRequest) (uuid.UUID, string, error) {
	transactionID, err := validateTransactionID(req.TransactionID)
	if err != nil {
		return uuid.Nil, "", err
	}

	code := normalizeOTP(req.Code)
	if code == "" {
		return uuid.Nil, "", apperror.NewBadRequest("code is required")
	}

	return transactionID, code, nil
}

func validateSetPasswordRequest(req setPasswordRequest) error {
	if req.Password == "" {
		return apperror.NewBadRequest("password is required")
	}

	return nil
}

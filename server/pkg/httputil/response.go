package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/leamout/leamout/pkg/apperror"
)

// Response is the standard JSON envelope for API responses.
type Response struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *ErrorObj `json:"error,omitempty"`
}

// ErrorObj describes an API error.
type ErrorObj struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OK sends a 200 response with data.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Created sends a 201 response with data.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// Accepted sends a 202 response for work accepted for asynchronous processing.
func Accepted(w http.ResponseWriter, data any) {
	JSON(w, http.StatusAccepted, Response{
		Success: true,
		Data:    data,
	})
}

// Partial sends a response that includes committed data plus an error.
func Partial(
	w http.ResponseWriter,
	status int,
	data any,
	err error,
) {
	errObj := &ErrorObj{
		Code:    "INTERNAL_ERROR",
		Message: "An unexpected error occurred",
	}

	if appErr, ok := errors.AsType[*apperror.AppError](err); ok {
		errObj.Code = appErr.Code
		errObj.Message = appErr.Message
	}

	JSON(w, status, Response{
		Success: false,
		Data:    data,
		Error:   errObj,
	})
}

// Error sends an error response based on AppError.
func Error(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	errObj := &ErrorObj{
		Code:    "INTERNAL_ERROR",
		Message: "An unexpected error occurred",
	}

	if appErr, ok := errors.AsType[*apperror.AppError](err); ok {
		status = appErr.Status
		errObj.Code = appErr.Code
		errObj.Message = appErr.Message
	}

	JSON(w, status, Response{
		Success: false,
		Error:   errObj,
	})
}

// JSON writes a JSON response.
func JSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}

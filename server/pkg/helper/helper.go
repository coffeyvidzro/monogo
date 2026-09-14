package helper

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/pkg/apperror"
)

func FormatTime(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}

	t := pgconv.TimestamptzToTime(value)
	if t.IsZero() {
		return ""
	}

	return t.Format(http.TimeFormat)
}

func DecodeJSON[T any](r *http.Request) (T, error) {
	var value T

	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		return value, apperror.NewBadRequest("invalid request body")
	}

	return value, nil
}

func ClientIP(r *http.Request) *string {
	value := r.RemoteAddr
	if value == "" {
		return nil
	}

	return &value
}

func UserAgent(r *http.Request) *string {
	value := r.Header.Get("User-Agent")
	if value == "" {
		return nil
	}

	return &value
}

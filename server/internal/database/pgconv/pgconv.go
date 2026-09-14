package pgconv

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDFromString converts a string to a PostgreSQL UUID.
func UUIDFromString(value string) (pgtype.UUID, error) {
	var id pgtype.UUID

	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse uuid: %w", err)
	}

	return id, nil
}

// UUIDToString converts a PostgreSQL UUID to a string.
// It returns an empty string if the UUID is invalid.
func UUIDToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}

	b := id.Bytes

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}

// TextFromString converts a string to PostgreSQL text.
// An empty string is treated as NULL.
func TextFromString(value string) pgtype.Text {
	return pgtype.Text{
		String: value,
		Valid:  true,
	}
}

// NullableText converts a nullable string to PostgreSQL text.
func NullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *value,
		Valid:  true,
	}
}

// TextToString converts PostgreSQL text to a string.
// It returns an empty string if the value is invalid.
func TextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

// String converts sqlc/PostgreSQL values to their string representation.
func String(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""

	case string:
		return typed

	case []byte:
		return string(typed)

	default:
		return fmt.Sprint(typed)
	}
}

// NullableString converts a PostgreSQL value to a nullable string.
func NullableString(value any) *string {
	if value == nil {
		return nil
	}

	converted := String(value)

	return &converted
}

// TimestamptzToTime converts PostgreSQL timestamptz to time.Time.
// It returns the zero time if the value is invalid.
func TimestamptzToTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}

	return value.Time
}

// NullableTimestamptz converts a nullable time.Time to PostgreSQL
// timestamptz. A nil or zero time is treated as NULL.
func NullableTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil || value.IsZero() {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{
		Time:  value.UTC(),
		Valid: true,
	}
}

// TimestamptzToTimePtr converts PostgreSQL timestamptz to *time.Time.
// It returns nil if the value is invalid.
func TimestamptzToTimePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	t := value.Time

	return &t
}

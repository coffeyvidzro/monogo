package freeswitch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) Channels(ctx context.Context) ([]Channel, error) {
	reply, err := c.Command(ctx, "show channels as json")
	if err != nil {
		return nil, err
	}
	return decodeRows(reply.Body, func(row map[string]any) Channel {
		return Channel{
			UUID:  stringField(row, "uuid"),
			Name:  stringField(row, "name"),
			State: stringField(row, "state"),
		}
	})
}

func (c *Client) Calls(ctx context.Context) ([]Call, error) {
	reply, err := c.Command(ctx, "show calls as json")
	if err != nil {
		return nil, err
	}
	return decodeRows(reply.Body, func(row map[string]any) Call {
		return Call{
			UUID:         stringField(row, "uuid"),
			CallerName:   stringField(row, "cid_name"),
			CallerNumber: stringField(row, "cid_num"),
			Destination:  stringField(row, "dest"),
			State:        stringField(row, "state"),
		}
	})
}

func (c *Client) Endpoints(ctx context.Context) ([]Endpoint, error) {
	reply, err := c.Command(ctx, "show endpoints as json")
	if err != nil {
		return nil, err
	}
	return decodeRows(reply.Body, func(row map[string]any) Endpoint {
		return Endpoint{
			Name: stringField(row, "name"),
			Type: stringField(row, "type"),
			Data: stringField(row, "data"),
		}
	})
}

func (c *Client) SofiaStatus(ctx context.Context, profile string) (SIPProfileStatus, error) {
	profile = strings.TrimSpace(profile)
	command := "sofia status"
	if profile != "" {
		command += " profile " + commandWord(profile)
	}

	reply, err := c.Command(ctx, command)
	if err != nil {
		return SIPProfileStatus{}, err
	}
	return SIPProfileStatus{
		Profile: profile,
		Raw:     reply.Body,
	}, nil
}

func decodeRows[T any](body string, mapRow func(map[string]any) T) ([]T, error) {
	rows, err := responseRows(body)
	if err != nil {
		return nil, err
	}

	result := make([]T, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapRow(row))
	}
	return result, nil
}

func responseRows(body string) ([]map[string]any, error) {
	body = strings.TrimSpace(body)

	var rows []map[string]any
	if err := json.Unmarshal([]byte(body), &rows); err == nil {
		return rows, nil
	}

	var envelope struct {
		RowCount *int             `json:"row_count"`
		Rows     []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return nil, fmt.Errorf("decode FreeSWITCH status response: %w", err)
	}
	if envelope.Rows == nil {
		if envelope.RowCount != nil && *envelope.RowCount == 0 {
			return []map[string]any{}, nil
		}
		return nil, fmt.Errorf("decode FreeSWITCH status response: missing rows field")
	}
	return envelope.Rows, nil
}

func stringField(row map[string]any, key string) string {
	value, ok := row[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

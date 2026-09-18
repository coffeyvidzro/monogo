package calls

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

type ChannelStore interface {
	Bind(context.Context, uuid.UUID, string) error
	Get(context.Context, uuid.UUID) (string, error)
	Delete(context.Context, uuid.UUID) error
}

type RedisChannelStore struct {
	client *redisintegration.Client
}

func NewRedisChannelStore(client *redisintegration.Client) *RedisChannelStore {
	if client == nil {
		panic("calls: Redis client is required")
	}
	return &RedisChannelStore{client: client}
}

func (s *RedisChannelStore) Bind(ctx context.Context, callID uuid.UUID, channelID string) error {
	channelID = strings.TrimSpace(channelID)
	if callID == uuid.Nil || channelID == "" {
		return fmt.Errorf("call id and channel id are required")
	}
	return s.client.Set(ctx, channelKey(callID), channelID, 24*time.Hour)
}

func (s *RedisChannelStore) Get(ctx context.Context, callID uuid.UUID) (string, error) {
	if callID == uuid.Nil {
		return "", fmt.Errorf("call id is required")
	}
	channelID, err := s.client.Get(ctx, channelKey(callID))
	if errors.Is(err, redisv9.Nil) {
		return "", ErrChannelUnavailable
	}
	if err != nil {
		return "", err
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", ErrChannelUnavailable
	}
	return channelID, nil
}

func (s *RedisChannelStore) Delete(ctx context.Context, callID uuid.UUID) error {
	if callID == uuid.Nil {
		return fmt.Errorf("call id is required")
	}
	return s.client.Delete(ctx, channelKey(callID))
}

func channelKey(callID uuid.UUID) string {
	return "telecom:calls:channel:" + callID.String()
}

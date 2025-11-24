package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	UserId    uuid.UUID
	SessionId uuid.UUID
	Timestamp int64
	Message   string
}

func streamName(userId uuid.UUID, sessionId uuid.UUID) string {
	return userId.String() + "-" + sessionId.String()
}

func (m *Message) streamName() string {
	return streamName(m.UserId, m.SessionId)
}

func streamID(timestamp int64) string {
	return fmt.Sprintf("%d-0", timestamp)
}

func (m *Message) streamID() string {
	return streamID(m.Timestamp)
}

type RedisClient struct {
	client *redis.Client
}

func CreateRedisClient(connectionString string) (*RedisClient, error) {
	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	return &RedisClient{
		client: client,
	}, nil
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) AddToStream(message *Message, streamTtl time.Duration) error {
	ctx := context.TODO()
	streamName := message.streamName()

	_, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]any{
			"message": message.Message,
		},
		ID: message.streamID(),
	}).Result()
	if err != nil {
		return err
	}

	err = r.client.Expire(ctx, streamName, streamTtl).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) Query(
	userId uuid.UUID,
	sessionId uuid.UUID,
	timestamp int64) (
	[]string, error) {

	ctx := context.TODO()

	streamName := streamName(userId, sessionId)
	startID := streamID(timestamp) // timestamp should be in ms
	endID := "+"

	res, err := r.client.XRange(ctx, streamName, startID, endID).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0, len(res))
	for _, msg := range res {
		if val, ok := msg.Values["message"].(string); ok {
			messages = append(messages, val)
		}
	}
	return messages, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

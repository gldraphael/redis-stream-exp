package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func CreateRedisClient() (*RedisClient, error) {
	opt, err := redis.ParseURL("redis://localhost:6379")
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ctx := context.TODO()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisClient{
		client: client,
	}, nil
}

func (r *RedisClient) AddToStream(message *Message) error {
	ctx := context.TODO()
	r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: message.StreamName(),
		Values: map[string]any{
			"message": message.Message,
		},
		ID: message.StreamID(),
	})
	return nil
}

func (r *RedisClient) Query(
	userId uuid.UUID,
	sessionId uuid.UUID,
	timestamp int64) (
	[]string, error) {

	ctx := context.TODO()

	streamName := userId.String() + "-" + sessionId.String()
	// startID := fmt.Sprintf("%d-0", timestamp) // timestamp should be in ms
	endID := "+"

	res, err := r.client.XRange(ctx, streamName, "-", endID).Result()
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

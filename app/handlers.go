package main

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ============================================================
// POST /message - Log a message
// ============================================================

type MessageRequest struct {
	Body struct {
		UserId    uuid.UUID `json:"userId" format:"uuid" doc:"User ID"`
		SessionId uuid.UUID `json:"sessionId" format:"uuid" doc:"Session ID"`
		Message   string    `json:"message" doc:"Message content"`
	}
}

type MessageResponse struct {
	Body struct {
		Timestamp int64 `json:"timestamp" doc:"Message timestamp in milliseconds"`
	}
}

// PostMessageHandler handles POST /message requests
func PostMessageHandler(redis *RedisClient) func(context.Context, *MessageRequest) (*MessageResponse, error) {
	return func(ctx context.Context, input *MessageRequest) (*MessageResponse, error) {
		timestamp := time.Now().UnixMilli()

		err := redis.AddToStream(&Message{
			UserId:    input.Body.UserId,
			SessionId: input.Body.SessionId,
			Timestamp: timestamp,
			Message:   input.Body.Message,
		}, time.Hour*1)

		if err != nil {
			log.Error().Err(err).Msg("Failed to add message to stream")
			return nil, huma.Error500InternalServerError("Failed to add message to stream")
		}

		resp := &MessageResponse{}
		resp.Body.Timestamp = timestamp

		log.Info().
			Str("userId", input.Body.UserId.String()).
			Str("sessionId", input.Body.SessionId.String()).
			Int64("timestamp", timestamp).
			Msg("Message logged")

		return resp, nil
	}
}

// ============================================================
// GET /message - Query messages
// ============================================================

type QueryRequest struct {
	UserId    string `query:"userId" format:"uuid" doc:"User ID"`
	SessionId string `query:"sessionId" format:"uuid" doc:"Session ID"`
	Timestamp int64  `query:"timestamp" doc:"Timestamp in milliseconds" minimum:"1" required:"true"`
}

type QueryResponse struct {
	Body struct {
		Messages []string `json:"messages" doc:"List of messages"`
	}
}

// GetMessagesHandler handles GET /message requests
func GetMessagesHandler(redis *RedisClient) func(context.Context, *QueryRequest) (*QueryResponse, error) {
	return func(ctx context.Context, input *QueryRequest) (*QueryResponse, error) {
		userId, err := uuid.Parse(input.UserId)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid userId format")
		}

		sessionId, err := uuid.Parse(input.SessionId)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid sessionId format")
		}

		messages, err := redis.Query(userId, sessionId, input.Timestamp)
		if err != nil {
			log.Error().Err(err).
				Str("userId", input.UserId).
				Str("sessionId", input.SessionId).
				Int64("timestamp", input.Timestamp).
				Msg("Failed to query messages")
			return nil, huma.Error500InternalServerError("Failed to query messages")
		}

		resp := &QueryResponse{}
		resp.Body.Messages = messages

		log.Info().
			Str("userId", input.UserId).
			Str("sessionId", input.SessionId).
			Int64("timestamp", input.Timestamp).
			Int("count", len(messages)).
			Msg("Messages queried")

		return resp, nil
	}
}

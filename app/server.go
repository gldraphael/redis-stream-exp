package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()

	// Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Create Redis client
	redis, err := CreateRedisClient(config.Redis.ConnectionString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to establish a redis connection")
	}

	// Create Chi router
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Create Huma API
	api := humachi.New(router, huma.DefaultConfig("redis-stream-exp", "0.1.0"))

	// Health check endpoints
	healthCheck := func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		if err := redis.Ping(ctx); err != nil {
			log.Error().Err(err).Msg("Health check failed")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","error":"` + err.Error() + `"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	router.Get("/livez", healthCheck)
	router.Get("/readyz", healthCheck)

	// POST /message endpoint - Log a message
	huma.Register(api, huma.Operation{
		OperationID: "post-message",
		Method:      http.MethodPost,
		Path:        "/message",
		Summary:     "Log a message",
		Description: "Adds a message to the Redis stream",
		Tags:        []string{"messages"},
	}, func(ctx context.Context, input *MessageRequest) (*MessageResponse, error) {
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
	})

	// GET /message endpoint - Query messages
	huma.Register(api, huma.Operation{
		OperationID: "get-messages",
		Method:      http.MethodGet,
		Path:        "/message",
		Summary:     "Query messages",
		Description: "Retrieves messages from a user session after a given timestamp",
		Tags:        []string{"messages"},
	}, func(ctx context.Context, input *QueryRequest) (*QueryResponse, error) {
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
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    config.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Str("port", config.Server.Port).Msg("Starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Forcefully shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Graceful shutdown failed")
	}

	// Close Redis connection
	if err := redis.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close the redis connection")
	}

	log.Info().Msg("Server shutdown complete")
}

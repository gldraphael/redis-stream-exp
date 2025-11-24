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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	config, err := LoadConfig(configPath)
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

	// Register message routes
	RegisterAppRoutes(api, redis)

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

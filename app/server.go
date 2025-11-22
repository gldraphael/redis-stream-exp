package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/ziflex/lecho/v3"
)

func main() {

	e := echo.New()
	e.HideBanner = true

	// Create and configure logger
	logger := lecho.New(
		os.Stdout,
		lecho.WithLevel(log.DEBUG),
		lecho.WithTimestamp(),
		lecho.WithCaller(),
	)
	e.Logger = logger

	// Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		e.Logger.Fatalf("Failed to load config. Error: %v", err)
	}

	// Add request ID middleware
	e.Use(middleware.RequestID())

	// Add lecho middleware for request logging
	e.Use(lecho.Middleware(lecho.Config{
		Logger:  logger,
		NestKey: "request",
	}))

	redis, err := CreateRedisClient(config.Redis.ConnectionString)
	if err != nil {
		e.Logger.Fatalf("Failed to establish a redis connection. Error: %v", err)
	}

	// Health check endpoints
	healthCheck := func(c echo.Context) error {
		ctx := context.Background()
		if err := redis.Ping(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
		}
		return c.NoContent(http.StatusOK)
	}
	e.GET("/livez", healthCheck)
	e.GET("/readyz", healthCheck)

	// Endpoint to log a message
	e.POST("/message", func(c echo.Context) error {

		type Request struct {
			UserId    uuid.UUID `json:"userId"    validate:"required"`
			SessionId uuid.UUID `json:"sessionId" validate:"required"`
			Message   string    `json:"message"   validate:"required"`
		}

		type Response struct {
			Timestamp int64 `json:"timestamp"`
		}

		msg := new(Request)
		if err := c.Bind(msg); err != nil {
			return err
		}

		timestamp := time.Now().UnixMilli()

		redis.AddToStream(&Message{
			UserId:    msg.UserId,
			SessionId: msg.SessionId,
			Timestamp: timestamp,
			Message:   msg.Message,
		}, time.Hour*1)

		return c.JSON(http.StatusAccepted, Response{
			Timestamp: timestamp,
		})
	})

	// Endpoint to query messages from a user session after a timestamp
	e.GET("/message", func(c echo.Context) error {

		type Request struct {
			UserId    uuid.UUID `query:"userId"    validate:"required"`
			SessionId uuid.UUID `query:"sessionId" validate:"required"`
			Timestamp int64     `query:"timestamp" validate:"required"`
		}

		type Response struct {
			Messages []string `json:"messages"`
		}

		var req Request
		if err := c.Bind(&req); err != nil {
			return err
		}
		if req.Timestamp == 0 {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "timestamp is required")
		}

		messages, err := redis.Query(req.UserId, req.SessionId, req.Timestamp)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, Response{
			Messages: messages,
		})
	})

	// Start server in a goroutine
	go func() {
		e.Logger.Info("Starting server")
		if err := e.Start(config.Server.Port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("Forcefully shutting down the server. Error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	e.Logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown the server
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatalf("Graceful shutdown failed. Error: %v", err)
	}

	// Close Redis connection
	if err := redis.Close(); err != nil {
		e.Logger.Errorf("Failed to close the redis connection. Error: %v", err)
	}

	e.Logger.Info("Server shutdown complete")
}

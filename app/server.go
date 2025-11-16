package main

import (
	"net/http"
	"os"
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

	// Add request ID middleware (optional but recommended)
	e.Use(middleware.RequestID())

	// Add lecho middleware for request logging
	e.Use(lecho.Middleware(lecho.Config{
		Logger:  logger,
		NestKey: "request",
	}))

	redis, err := CreateRedisClient()
	if err != nil {
		e.Logger.Fatalf("Failed to establish a redis connection. Error: %s", err.Error())
	}

	// Endpoint to log a message
	e.POST("/message", func(c echo.Context) error {

		type PostMessageRequest struct {
			UserId    uuid.UUID `json:"userId"    validate:"required"`
			SessionId uuid.UUID `json:"sessionId" validate:"required"`
			Message   string    `json:"message"   validate:"required"`
		}

		type PostMessageResponse struct {
			Timestamp int64 `json:"timestamp"`
		}

		msg := new(PostMessageRequest)
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

		return c.JSON(http.StatusAccepted, PostMessageResponse{
			Timestamp: timestamp,
		})
	})

	// Endpoint to query messages from a user session after a timestamp
	e.GET("/message", func(c echo.Context) error {

		type GetMessageRequest struct {
			UserId    uuid.UUID `query:"userId"    validate:"required"`
			SessionId uuid.UUID `query:"sessionId" validate:"required"`
			Timestamp int64     `query:"timestamp" validate:"required"`
		}

		var req GetMessageRequest
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
		return c.JSON(http.StatusOK, messages)
	})

	e.Logger.Fatal(e.Start(":1323"))
}

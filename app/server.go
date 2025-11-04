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

type (
	MessageRequest struct {
		UserId    uuid.UUID `json:"userId"    validate:"required"`
		SessionId uuid.UUID `json:"sessionId" validate:"required"`
		Message   string    `json:"message"   validate:"required"`
	}

	ErrorResponse struct {
		Message string `json:"message"`
	}
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

	// Endpoint to log an event
	e.POST("/message", func(c echo.Context) error {
		msg := new(MessageRequest)
		if err := c.Bind(msg); err != nil {
			return err
		}

		redis.AddToStream(&Message{
			UserId:    msg.UserId,
			SessionId: msg.SessionId,
			Timestamp: time.Now().UnixMilli(),
			Message:   msg.Message,
		})
		return c.NoContent(http.StatusAccepted)
	})

	// Endpoint to query relevent logs
	e.GET("/message", func(c echo.Context) error {

		userId, err := uuid.Parse(c.QueryParam("userId"))
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Message: "Query string userId is required."})
		}

		sessionId, err := uuid.Parse(c.QueryParam("sessionId"))
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Message: "Query string sessionId is required."})
		}

		e.Logger.Debugf("%s %s", userId.String(), sessionId.String())
		oneHourAgo := time.Now().Add(-time.Hour)
		messages, err := redis.Query(userId, sessionId, oneHourAgo.UnixMilli())
		return c.JSON(http.StatusOK, messages)
	})

	e.Logger.Fatal(e.Start(":1323"))
}

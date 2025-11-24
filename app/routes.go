package main

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterAppRoutes registers all application routes
func RegisterAppRoutes(api huma.API, redis *RedisClient) {
	// POST /message endpoint - Log a message
	huma.Register(api, huma.Operation{
		OperationID: "post-message",
		Method:      http.MethodPost,
		Path:        "/message",
		Summary:     "Log a message",
		Description: "Adds a message to the Redis stream",
		Tags:        []string{"messages"},
	}, PostMessageHandler(redis))

	// GET /message endpoint - Query messages
	huma.Register(api, huma.Operation{
		OperationID: "get-messages",
		Method:      http.MethodGet,
		Path:        "/message",
		Summary:     "Query messages",
		Description: "Retrieves messages from a user session after a given timestamp",
		Tags:        []string{"messages"},
	}, GetMessagesHandler(redis))
}

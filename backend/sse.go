package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	gameID string
	userID string
	events chan string
}

// SSEEvent represents an event sent to clients
type SSEEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// StreamGameUpdates handles SSE connections for game updates
func StreamGameUpdates(w http.ResponseWriter, r *http.Request, gameID, userID string, redisClient *redis.Client) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create context that cancels when client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Subscribe to Redis channel for this game
	pubsub := redisClient.Subscribe(ctx, fmt.Sprintf("game:%s", gameID))
	defer pubsub.Close()

	log.Printf("[SSE] User %s connected to game %s stream", userID, gameID)

	// Send initial connection message
	sendSSEMessage(w, flusher, "connected", map[string]string{
		"message": "Connected to game stream",
		"gameId":  gameID,
	})

	// Keepalive ticker
	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	// Listen for messages
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[SSE] User %s disconnected from game %s", userID, gameID)
			return
		case msg := <-ch:
			// Forward Redis message to client
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			flusher.Flush()
		case <-keepalive.C:
			// Send keepalive
			sendSSEMessage(w, flusher, "keepalive", map[string]int64{
				"timestamp": time.Now().Unix(),
			})
		}
	}
}

// PublishGameEvent publishes an event to all clients watching a game
func PublishGameEvent(redisClient *redis.Client, gameID, eventType string, payload interface{}) error {
	event := SSEEvent{
		Type:    eventType,
		Payload: payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	ctx := context.Background()
	return redisClient.Publish(ctx, fmt.Sprintf("game:%s", gameID), string(data)).Err()
}

// sendSSEMessage sends a formatted SSE message
func sendSSEMessage(w http.ResponseWriter, flusher http.Flusher, eventType string, payload interface{}) {
	event := SSEEvent{
		Type:    eventType,
		Payload: payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[SSE] Error marshaling message: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", string(data))
	flusher.Flush()
}

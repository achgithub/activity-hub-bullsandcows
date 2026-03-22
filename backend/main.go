package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/achgithub/activity-hub-auth"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var (
	db          *sql.DB
	identityDB  *sql.DB
	redisClient *redis.Client
)

func main() {
	// Get socket path from environment or use default
	socketPath := os.Getenv("SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/tmp/activity-hub-bullsandcows.sock"
	}

	// Remove existing socket if it exists
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatalf("Failed to remove existing socket: %v", err)
	}

	// Get database host from environment or use default
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	// Get Redis address from environment or use default
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	// Initialize Bulls and Cows database connection
	connStr := fmt.Sprintf("host=%s port=5555 user=activityhub password=pubgames dbname=bullsandcows sslmode=disable", dbHost)
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Printf("✅ Connected to PostgreSQL database: bullsandcows")

	// Connect to Activity Hub identity database for auth
	identityConnStr := fmt.Sprintf("host=%s port=5555 user=activityhub password=pubgames dbname=activity_hub sslmode=disable", dbHost)
	identityDB, err = sql.Open("postgres", identityConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to identity database: %v", err)
	}
	defer identityDB.Close()

	if err := identityDB.Ping(); err != nil {
		log.Fatalf("Failed to ping identity database: %v", err)
	}
	log.Printf("✅ Connected to identity database: activity_hub")

	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("✅ Connected to Redis at %s", redisAddr)

	// Build auth middleware using activity-hub-auth SDK
	authMiddleware := auth.Middleware(identityDB)
	sseMiddleware := auth.SSEMiddleware(identityDB)

	// Create router
	r := mux.NewRouter()

	// Public endpoints
	r.HandleFunc("/api/config", GetConfig).Methods("GET")

	// SSE endpoint (uses query-param auth)
	r.Handle("/api/game/{gameId}/stream", sseMiddleware(http.HandlerFunc(StreamGame(redisClient)))).Methods("GET")

	// Authenticated endpoints
	r.Handle("/api/game", authMiddleware(http.HandlerFunc(CreateGame(db, redisClient)))).Methods("POST")
	r.Handle("/api/game/{gameId}", authMiddleware(http.HandlerFunc(GetGame(db)))).Methods("GET")
	r.Handle("/api/game/{gameId}/set-code", authMiddleware(http.HandlerFunc(SetCode(db, redisClient)))).Methods("POST")
	r.Handle("/api/game/{gameId}/guess", authMiddleware(http.HandlerFunc(MakeGuess(db, redisClient)))).Methods("POST")

	// Serve static files (React build)
	staticDir := "./static"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Printf("Warning: Static directory not found at %s", staticDir)
	} else {
		// Serve index.html for root
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})

		// Serve other static files
		r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(staticDir))))
	}

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to create Unix socket: %v", err)
	}
	defer listener.Close()

	// Set socket permissions
	if err := os.Chmod(socketPath, 0777); err != nil {
		log.Fatalf("Failed to set socket permissions: %v", err)
	}

	// Start server on Unix socket
	log.Printf("🎯 Bulls and Cows server starting on Unix socket: %s", socketPath)
	log.Fatal(http.Serve(listener, handler))
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	authlib "github.com/achgithub/activity-hub-auth"
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

	// Get database configuration from environment (match Activity Hub defaults)
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "activityhub")
	dbPass := getEnv("DB_PASS", "pubgames")
	dbName := getEnv("DB_NAME", "bullsandcows")
	identityDBName := getEnv("IDENTITY_DB_NAME", "activity_hub")

	// Get Redis configuration
	redisHost := getEnv("REDIS_HOST", "127.0.0.1")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	// Initialize Bulls and Cows database connection
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	))
	if err != nil {
		log.Fatalf("Failed to connect to Bulls and Cows database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping Bulls and Cows database: %v", err)
	}
	log.Printf("✅ Connected to PostgreSQL database: %s", dbName)

	// Connect to Activity Hub identity database for auth
	identityDB, err = sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, identityDBName,
	))
	if err != nil {
		log.Fatalf("Failed to connect to identity database: %v", err)
	}
	defer identityDB.Close()

	if err := identityDB.Ping(); err != nil {
		log.Fatalf("Failed to ping identity database: %v", err)
	}
	log.Printf("✅ Connected to identity database: %s", identityDBName)

	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("✅ Connected to Redis at %s:%s", redisHost, redisPort)

	// Build auth middleware using activity-hub-auth SDK
	authMiddleware := authlib.Middleware(identityDB)

	// Create router
	r := mux.NewRouter()

	// Public endpoints
	r.HandleFunc("/api/health", HandleHealth).Methods("GET")
	r.HandleFunc("/api/config", GetConfig).Methods("GET")

	// SSE endpoint (handles query-param auth internally)
	r.HandleFunc("/api/game/{gameId}/stream", StreamGame(redisClient, identityDB)).Methods("GET")

	// Authenticated endpoints
	r.Handle("/api/game", authMiddleware(http.HandlerFunc(CreateGame(db, redisClient)))).Methods("POST")
	r.Handle("/api/game/{gameId}", authMiddleware(http.HandlerFunc(GetGame(db)))).Methods("GET")
	r.Handle("/api/game/{gameId}/set-code", authMiddleware(http.HandlerFunc(SetCode(db, redisClient)))).Methods("POST")
	r.Handle("/api/game/{gameId}/guess", authMiddleware(http.HandlerFunc(MakeGuess(db, redisClient)))).Methods("POST")

	// Serve static files (React build)
	staticPath := getEnv("STATIC_PATH", "./static")
	if _, err := os.Stat(staticPath); os.IsNotExist(err) {
		log.Printf("Warning: Static directory not found at %s", staticPath)
	} else {
		r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticPath)))
		log.Printf("📁 Serving static files from: %s", staticPath)
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

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

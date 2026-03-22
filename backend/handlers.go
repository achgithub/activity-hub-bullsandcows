package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	authlib "github.com/achgithub/activity-hub-common/auth"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetConfig returns app configuration for the identity shell
func GetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConfigResponse{
		AppName:    "Bulls and Cows",
		MinPlayers: 0,
		MaxPlayers: 2,
		GameOptions: []GameOption{
			{
				ID:      "mode",
				Label:   "Mode",
				Type:    "select",
				Default: "colors",
				Options: []OptionValue{
					{Value: "colors", Label: "Colors (4 pegs, 6 colors)"},
					{Value: "numbers", Label: "Numbers (5 digits, 0-9)"},
				},
			},
		},
	})
}

// CreateGame creates a new game (solo or 2-player)
func CreateGame(db *sql.DB, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authlib.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := user.Email

		var req CreateGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Detect if this is a 2-player challenge request (from identity-shell)
		isChallenge := req.Player1ID != "" && req.Player2ID != ""

		// Apply default mode if missing or invalid
		if req.Mode != "colors" && req.Mode != "numbers" {
			req.Mode = "colors"
		}

		// Always generate a new UUID for gameID (database requires UUID type)
		gameID := uuid.New().String()

		// Set max guesses based on mode
		maxGuesses := 12
		if req.Mode == "numbers" {
			maxGuesses = 25
		}

		var game Game
		var createdAt, updatedAt time.Time

		if isChallenge {
			// 2-player challenge: dual-code simultaneous mode
			// Both players must set their codes before gameplay begins
			log.Printf("Creating 2-player dual-code game: %s vs %s", req.Player1Name, req.Player2Name)

			query := `
				INSERT INTO games (id, mode, variant, player1_id, player2_id, max_guesses, status, current_turn)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING created_at, updated_at
			`
			err := db.QueryRow(query, gameID, req.Mode, "2player", req.Player1ID, req.Player2ID, maxGuesses, "code_setting", 0).
				Scan(&createdAt, &updatedAt)
			if err != nil {
				log.Printf("Error creating 2-player game: %v", err)
				http.Error(w, "Failed to create game", http.StatusInternalServerError)
				return
			}

			game = Game{
				ID:             gameID,
				Mode:           req.Mode,
				Variant:        "2player",
				Player1ID:      req.Player1ID,
				Player2ID:      req.Player2ID,
				Player1CodeSet: false,
				Player2CodeSet: false,
				CurrentTurn:    0,
				MaxGuesses:     maxGuesses,
				Status:         "code_setting",
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
				Guesses:        []Guess{},
			}
		} else {
			// Solo play: AI-generated code, traditional gameplay
			if req.Variant != "1player" {
				http.Error(w, "Only 1player variant supported for solo play", http.StatusBadRequest)
				return
			}

			secretCode := GenerateSecretCode(req.Mode)

			query := `
				INSERT INTO games (id, mode, variant, secret_code, code_breaker, max_guesses, status)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING created_at, updated_at
			`
			err := db.QueryRow(query, gameID, req.Mode, "1player", secretCode, userID, maxGuesses, "active").
				Scan(&createdAt, &updatedAt)
			if err != nil {
				log.Printf("Error creating solo game: %v", err)
				http.Error(w, "Failed to create game", http.StatusInternalServerError)
				return
			}

			game = Game{
				ID:          gameID,
				Mode:        req.Mode,
				Variant:     "1player",
				CodeBreaker: userID,
				MaxGuesses:  maxGuesses,
				Status:      "active",
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				Guesses:     []Guess{},
			}
		}

		// Store in Redis for quick access (1 hour TTL)
		ctx := context.Background()
		gameJSON, _ := json.Marshal(game)
		redisClient.Set(ctx, fmt.Sprintf("game:%s", gameID), gameJSON, time.Hour)

		// Publish game created event
		PublishGameEvent(redisClient, gameID, "game_created", game)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"gameId":  game.ID,
			"game":    game,
		})
	}
}

// GetGame retrieves a game by ID
func GetGame(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authlib.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := user.Email

		vars := mux.Vars(r)
		gameID := vars["gameId"]

		// Fetch game from database
		var game Game
		var winner, secretCode, codeBreaker sql.NullString
		var player1ID, player2ID, player1Code, player2Code sql.NullString
		var player1CodeSet, player2CodeSet sql.NullBool
		var completedAt sql.NullTime

		query := `
			SELECT id, mode, variant, status, max_guesses, current_turn,
			       winner, created_at, updated_at, completed_at,
			       secret_code, code_breaker,
			       player1_id, player2_id, player1_code, player2_code, player1_code_set, player2_code_set
			FROM games WHERE id = $1
		`
		err := db.QueryRow(query, gameID).Scan(
			&game.ID, &game.Mode, &game.Variant, &game.Status, &game.MaxGuesses, &game.CurrentTurn,
			&winner, &game.CreatedAt, &game.UpdatedAt, &completedAt,
			&secretCode, &codeBreaker,
			&player1ID, &player2ID, &player1Code, &player2Code, &player1CodeSet, &player2CodeSet,
		)
		if err == sql.ErrNoRows {
			http.Error(w, "Game not found", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("Error fetching game: %v", err)
			http.Error(w, "Failed to fetch game", http.StatusInternalServerError)
			return
		}

		// Populate nullable fields
		if winner.Valid {
			game.Winner = &winner.String
		}
		if completedAt.Valid {
			game.CompletedAt = &completedAt.Time
		}
		if secretCode.Valid {
			game.SecretCode = secretCode.String
		}
		if codeBreaker.Valid {
			game.CodeBreaker = codeBreaker.String
		}
		if player1ID.Valid {
			game.Player1ID = player1ID.String
		}
		if player2ID.Valid {
			game.Player2ID = player2ID.String
		}
		if player1Code.Valid {
			game.Player1Code = player1Code.String
		}
		if player2Code.Valid {
			game.Player2Code = player2Code.String
		}
		if player1CodeSet.Valid {
			game.Player1CodeSet = player1CodeSet.Bool
		}
		if player2CodeSet.Valid {
			game.Player2CodeSet = player2CodeSet.Bool
		}

		// Verify user is part of this game
		if game.Variant == "1player" {
			if game.CodeBreaker != userID {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}
		} else if game.Variant == "2player" {
			if game.Player1ID != userID && game.Player2ID != userID {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}
		}

		// Fetch guesses
		guessQuery := `
			SELECT id, game_id, turn_number, player_id, guess_code, bulls, cows, guessed_at
			FROM guesses
			WHERE game_id = $1
			ORDER BY turn_number ASC, player_id ASC
		`
		rows, err := db.Query(guessQuery, gameID)
		if err != nil {
			log.Printf("Error fetching guesses: %v", err)
			http.Error(w, "Failed to fetch guesses", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		game.Guesses = []Guess{}
		for rows.Next() {
			var guess Guess
			err := rows.Scan(&guess.ID, &guess.GameID, &guess.TurnNumber, &guess.PlayerID, &guess.GuessCode, &guess.Bulls, &guess.Cows, &guess.GuessedAt)
			if err != nil {
				log.Printf("Error scanning guess: %v", err)
				continue
			}
			game.Guesses = append(game.Guesses, guess)
		}

		// Hide secret codes if game is still active
		if game.Status == "active" || game.Status == "code_setting" {
			game.SecretCode = ""
			// For 2-player, only hide opponent's code
			if game.Variant == "2player" {
				if game.Player1ID != userID {
					game.Player1Code = ""
				}
				if game.Player2ID != userID {
					game.Player2Code = ""
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(game)
	}
}

// StreamGame handles SSE connections for game updates
func StreamGame(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authlib.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		gameID := vars["gameId"]

		StreamGameUpdates(w, r, gameID, user.Email, redisClient)
	}
}

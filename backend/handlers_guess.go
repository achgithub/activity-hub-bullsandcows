package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	authlib "github.com/achgithub/activity-hub-common/auth"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// MakeGuess handles a guess submission (solo or 2-player)
func MakeGuess(db *sql.DB, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authlib.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := user.Email

		vars := mux.Vars(r)
		gameID := vars["gameId"]

		var req MakeGuessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Fetch game - different fields for solo vs 2-player
		var game Game
		query := `
			SELECT id, mode, variant, status, max_guesses, current_turn,
			       secret_code, code_breaker,
			       player1_id, player2_id, player1_code, player2_code, player1_code_set, player2_code_set
			FROM games WHERE id = $1
		`
		var secretCode, codeBreaker sql.NullString
		var player1ID, player2ID, player1Code, player2Code sql.NullString
		var player1CodeSet, player2CodeSet sql.NullBool

		err := db.QueryRow(query, gameID).Scan(
			&game.ID, &game.Mode, &game.Variant, &game.Status, &game.MaxGuesses, &game.CurrentTurn,
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

		// Populate game struct
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

		// Check if game is active
		if game.Status != "active" {
			http.Error(w, "Game is not active", http.StatusBadRequest)
			return
		}

		// Validate guess
		guess := strings.ToUpper(req.Guess)
		if err := ValidateGuess(guess, game.Mode); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Route to appropriate handler based on variant
		if game.Variant == "1player" {
			handleSoloGuess(w, db, redisClient, &game, userID, guess)
		} else if game.Variant == "2player" {
			handleTwoPlayerGuess(w, db, redisClient, &game, userID, guess)
		} else {
			http.Error(w, "Unknown game variant", http.StatusBadRequest)
		}
	}
}

// handleSoloGuess processes a guess for solo play (original logic)
func handleSoloGuess(w http.ResponseWriter, db *sql.DB, redisClient *redis.Client, game *Game, userID, guess string) {
	gameID := game.ID

	// Verify user is the code breaker
	if game.CodeBreaker != userID {
		http.Error(w, "Only the code breaker can make guesses", http.StatusForbidden)
		return
	}

	// Count existing guesses for this player
	var guessCount int
	db.QueryRow("SELECT COUNT(*) FROM guesses WHERE game_id = $1 AND player_id = $2", gameID, userID).Scan(&guessCount)

	if guessCount >= game.MaxGuesses {
		http.Error(w, "Maximum guesses reached", http.StatusBadRequest)
		return
	}

	// Calculate bulls and cows
	bulls, cows := CalculateBullsAndCows(game.SecretCode, guess)

	// Save guess
	turnNumber := guessCount + 1
	var guessID int
	var guessedAt time.Time
	insertQuery := `
		INSERT INTO guesses (game_id, turn_number, player_id, guess_code, bulls, cows)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, guessed_at
	`
	err := db.QueryRow(insertQuery, gameID, turnNumber, userID, guess, bulls, cows).Scan(&guessID, &guessedAt)
	if err != nil {
		log.Printf("Error saving guess: %v", err)
		http.Error(w, "Failed to save guess", http.StatusInternalServerError)
		return
	}

	// Check for win or loss
	var newStatus string
	var winner *string
	var completedAt *time.Time
	now := time.Now()

	if CheckWin(bulls, game.Mode) {
		newStatus = "won"
		winner = &userID
		completedAt = &now
	} else if turnNumber >= game.MaxGuesses {
		newStatus = "lost"
		completedAt = &now
	} else {
		newStatus = "active"
	}

	// Update game status if changed
	if newStatus != "active" {
		updateQuery := `UPDATE games SET status = $1, winner = $2, completed_at = $3 WHERE id = $4`
		_, err = db.Exec(updateQuery, newStatus, winner, completedAt, gameID)
		if err != nil {
			log.Printf("Error updating game status: %v", err)
		}
	}

	// Create response
	guessResponse := Guess{
		ID:         guessID,
		GameID:     gameID,
		TurnNumber: turnNumber,
		PlayerID:   userID,
		GuessCode:  guess,
		Bulls:      bulls,
		Cows:       cows,
		GuessedAt:  guessedAt,
	}

	// Publish SSE event
	eventPayload := map[string]interface{}{
		"guess":  guessResponse,
		"status": newStatus,
	}
	if newStatus != "active" {
		eventPayload["secretCode"] = game.SecretCode
		eventPayload["winner"] = winner
	}
	PublishGameEvent(redisClient, gameID, "guess_made", eventPayload)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"guess":      guessResponse,
		"status":     newStatus,
		"secretCode": func() string { if newStatus != "active" { return game.SecretCode }; return "" }(),
	})
}

// handleTwoPlayerGuess processes a guess for 2-player turn-based mode
func handleTwoPlayerGuess(w http.ResponseWriter, db *sql.DB, redisClient *redis.Client, game *Game, userID, guess string) {
	gameID := game.ID

	// Verify user is a player in this game
	if game.Player1ID != userID && game.Player2ID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if both codes are set
	if !game.Player1CodeSet || !game.Player2CodeSet {
		http.Error(w, "Both players must set codes before guessing", http.StatusBadRequest)
		return
	}

	currentTurn := game.CurrentTurn
	if currentTurn == 0 {
		http.Error(w, "Game has not started yet", http.StatusBadRequest)
		return
	}

	// Check if this player has already guessed for the current turn
	var existingGuess int
	db.QueryRow("SELECT COUNT(*) FROM guesses WHERE game_id = $1 AND turn_number = $2 AND player_id = $3",
		gameID, currentTurn, userID).Scan(&existingGuess)

	if existingGuess > 0 {
		http.Error(w, "You have already guessed for this turn", http.StatusBadRequest)
		return
	}

	// Determine which code this player is trying to crack
	var opponentCode string
	if game.Player1ID == userID {
		opponentCode = game.Player2Code
	} else {
		opponentCode = game.Player1Code
	}

	// Calculate bulls and cows against opponent's code
	bulls, cows := CalculateBullsAndCows(opponentCode, guess)

	// Save this player's guess
	var guessID int
	var guessedAt time.Time
	insertQuery := `
		INSERT INTO guesses (game_id, turn_number, player_id, guess_code, bulls, cows)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, guessed_at
	`
	err := db.QueryRow(insertQuery, gameID, currentTurn, userID, guess, bulls, cows).Scan(&guessID, &guessedAt)
	if err != nil {
		log.Printf("Error saving guess: %v", err)
		http.Error(w, "Failed to save guess", http.StatusInternalServerError)
		return
	}

	// Check if both players have now guessed for this turn
	var bothGuessed bool
	db.QueryRow(`
		SELECT COUNT(DISTINCT player_id) = 2
		FROM guesses
		WHERE game_id = $1 AND turn_number = $2
	`, gameID, currentTurn).Scan(&bothGuessed)

	// Create response
	guessResponse := Guess{
		ID:         guessID,
		GameID:     gameID,
		TurnNumber: currentTurn,
		PlayerID:   userID,
		GuessCode:  guess,
		Bulls:      bulls,
		Cows:       cows,
		GuessedAt:  guessedAt,
	}

	if bothGuessed {
		// Both players have guessed - evaluate the turn and check for win/draw
		evaluateTurn(db, redisClient, game, currentTurn)

		// Re-fetch game status
		var newStatus string
		var winner sql.NullString
		db.QueryRow("SELECT status, winner FROM games WHERE id = $1", gameID).Scan(&newStatus, &winner)

		// Publish turn_complete event
		PublishGameEvent(redisClient, gameID, "turn_complete", map[string]interface{}{
			"gameId": gameID,
			"turn":   currentTurn,
			"status": newStatus,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"guess":       guessResponse,
			"bothGuessed": true,
			"status":      newStatus,
			"turnComplete": true,
		})
	} else {
		// Waiting for opponent to guess
		PublishGameEvent(redisClient, gameID, "guess_submitted", map[string]interface{}{
			"gameId":  gameID,
			"turn":    currentTurn,
			"player":  userID,
			"waiting": true,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"guess":       guessResponse,
			"bothGuessed": false,
			"waiting":     true,
		})
	}
}

// evaluateTurn checks for win/draw/continue after both players have guessed
func evaluateTurn(db *sql.DB, redisClient *redis.Client, game *Game, turnNumber int) {
	gameID := game.ID

	// Fetch both guesses for this turn
	rows, err := db.Query(`
		SELECT player_id, bulls, cows
		FROM guesses
		WHERE game_id = $1 AND turn_number = $2
	`, gameID, turnNumber)
	if err != nil {
		log.Printf("Error fetching turn guesses: %v", err)
		return
	}
	defer rows.Close()

	var player1Bulls, player2Bulls int
	for rows.Next() {
		var playerID string
		var bulls, cows int
		rows.Scan(&playerID, &bulls, &cows)

		if playerID == game.Player1ID {
			player1Bulls = bulls
		} else if playerID == game.Player2ID {
			player2Bulls = bulls
		}
	}

	// Check for wins
	codeLength := 4
	if game.Mode == "numbers" {
		codeLength = 5
	}

	player1Won := player1Bulls == codeLength
	player2Won := player2Bulls == codeLength

	var newStatus string
	var winner *string
	now := time.Now()

	if player1Won && player2Won {
		// Draw - both cracked on same turn
		newStatus = "draw"
		drawStr := "draw"
		winner = &drawStr
		log.Printf("Game %s: Draw - both players won on turn %d", gameID, turnNumber)
	} else if player1Won {
		// Player 1 wins
		newStatus = "won"
		winner = &game.Player1ID
		log.Printf("Game %s: Player 1 wins on turn %d", gameID, turnNumber)
	} else if player2Won {
		// Player 2 wins
		newStatus = "won"
		winner = &game.Player2ID
		log.Printf("Game %s: Player 2 wins on turn %d", gameID, turnNumber)
	} else if turnNumber >= game.MaxGuesses {
		// Draw - both exceeded max guesses
		newStatus = "draw"
		drawStr := "draw"
		winner = &drawStr
		log.Printf("Game %s: Draw - max guesses reached", gameID)
	} else {
		// Game continues - advance to next turn
		newTurn := turnNumber + 1
		db.Exec("UPDATE games SET current_turn = $1 WHERE id = $2", newTurn, gameID)
		log.Printf("Game %s: Advancing to turn %d", gameID, newTurn)
		return
	}

	// Game over - update status
	db.Exec("UPDATE games SET status = $1, winner = $2, completed_at = $3 WHERE id = $4",
		newStatus, winner, now, gameID)
}

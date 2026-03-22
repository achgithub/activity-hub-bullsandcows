package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	authlib "github.com/achgithub/activity-hub-auth"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// SetCode allows a player to set their secret code in a 2-player game
func SetCode(db *sql.DB, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := authlib.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := user.Email

		vars := mux.Vars(r)
		gameID := vars["gameId"]

		var req SetCodeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Fetch game from database
		var game Game
		var player1CodeSet, player2CodeSet bool
		var player1Code, player2Code sql.NullString
		query := `
			SELECT id, mode, variant, status, player1_id, player2_id, player1_code, player1_code_set, player2_code, player2_code_set, max_guesses, current_turn
			FROM games
			WHERE id = $1
		`
		err := db.QueryRow(query, gameID).Scan(
			&game.ID, &game.Mode, &game.Variant, &game.Status,
			&game.Player1ID, &game.Player2ID, &player1Code, &player1CodeSet, &player2Code, &player2CodeSet,
			&game.MaxGuesses, &game.CurrentTurn,
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

		// Verify this is a 2-player game
		if game.Variant != "2player" {
			http.Error(w, "Only 2-player games require code setting", http.StatusBadRequest)
			return
		}

		// Verify user is a player in this game
		if game.Player1ID != userID && game.Player2ID != userID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Verify game is in code_setting status
		if game.Status != "code_setting" {
			http.Error(w, "Game is not in code setting phase", http.StatusBadRequest)
			return
		}

		// Validate the code
		code := strings.ToUpper(req.Code)
		if err := ValidateGuess(code, game.Mode); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Determine which player is setting the code
		isPlayer1 := game.Player1ID == userID
		var updateQuery string
		if isPlayer1 {
			if player1CodeSet {
				http.Error(w, "You have already set your code", http.StatusBadRequest)
				return
			}
			updateQuery = `UPDATE games SET player1_code = $1, player1_code_set = true WHERE id = $2`
		} else {
			if player2CodeSet {
				http.Error(w, "You have already set your code", http.StatusBadRequest)
				return
			}
			updateQuery = `UPDATE games SET player2_code = $1, player2_code_set = true WHERE id = $2`
		}

		// Set the code
		_, err = db.Exec(updateQuery, code, gameID)
		if err != nil {
			log.Printf("Error setting code: %v", err)
			http.Error(w, "Failed to set code", http.StatusInternalServerError)
			return
		}

		// Check if both players have now set their codes
		var bothCodesSet bool
		db.QueryRow(`SELECT player1_code_set AND player2_code_set FROM games WHERE id = $1`, gameID).Scan(&bothCodesSet)

		if bothCodesSet {
			// Both codes set - start the game!
			_, err = db.Exec(`UPDATE games SET status = 'active', current_turn = 1 WHERE id = $1`, gameID)
			if err != nil {
				log.Printf("Error starting game: %v", err)
			}

			log.Printf("Both codes set for game %s - game starting!", gameID)

			// Publish game_started event to both players
			PublishGameEvent(redisClient, gameID, "both_codes_set", map[string]interface{}{
				"gameId": gameID,
				"status": "active",
			})
		} else {
			// One code set, waiting for the other
			log.Printf("Code set for game %s - waiting for other player", gameID)

			// Publish code_set event
			PublishGameEvent(redisClient, gameID, "code_set", map[string]interface{}{
				"gameId":  gameID,
				"player":  userID,
				"waiting": true,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"codeSet":      true,
			"bothCodesSet": bothCodesSet,
			"status":       func() string { if bothCodesSet { return "active" } else { return "code_setting" } }(),
		})
	}
}

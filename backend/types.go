package main

import "time"

// Game represents a Bulls and Cows game
type Game struct {
	ID         string     `json:"id"`
	Mode       string     `json:"mode"`
	Variant    string     `json:"variant"`
	MaxGuesses int        `json:"maxGuesses"`
	Status     string     `json:"status"`
	Winner     *string    `json:"winner,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Solo play fields (variant='1player')
	SecretCode  string `json:"secretCode,omitempty"`
	CodeBreaker string `json:"codeBreaker,omitempty"`

	// 2-player fields (variant='2player')
	Player1ID      string `json:"player1Id,omitempty"`
	Player1Code    string `json:"player1Code,omitempty"`
	Player1CodeSet bool   `json:"player1CodeSet"`
	Player2ID      string `json:"player2Id,omitempty"`
	Player2Code    string `json:"player2Code,omitempty"`
	Player2CodeSet bool   `json:"player2CodeSet"`
	CurrentTurn    int    `json:"currentTurn"`

	Guesses []Guess `json:"guesses,omitempty"`
}

// Guess represents a single guess in the game
type Guess struct {
	ID         int       `json:"id"`
	GameID     string    `json:"gameId"`
	TurnNumber int       `json:"turnNumber"`
	PlayerID   string    `json:"playerId"`
	GuessCode  string    `json:"guessCode"`
	Bulls      int       `json:"bulls"`
	Cows       int       `json:"cows"`
	GuessedAt  time.Time `json:"guessedAt"`
}

// CreateGameRequest represents the request to create a new game
type CreateGameRequest struct {
	// Solo play fields
	Mode    string `json:"mode"`    // "colors" or "numbers"
	Variant string `json:"variant"` // "1player" or "2player"

	// 2-player challenge fields (from identity-shell)
	ChallengeID string `json:"challengeId"`
	Player1ID   string `json:"player1Id"`
	Player1Name string `json:"player1Name"`
	Player2ID   string `json:"player2Id"`
	Player2Name string `json:"player2Name"`
}

// SetCodeRequest represents a request to set secret code for 2-player
type SetCodeRequest struct {
	Code string `json:"code"`
}

// MakeGuessRequest represents a guess submission
type MakeGuessRequest struct {
	Guess string `json:"guess"`
}

// ConfigResponse represents the app configuration
type ConfigResponse struct {
	AppName     string       `json:"appName"`
	MinPlayers  int          `json:"minPlayers"`
	MaxPlayers  int          `json:"maxPlayers"`
	GameOptions []GameOption `json:"gameOptions"`
}

// GameOption represents a configurable game option
type GameOption struct {
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Type    string        `json:"type"`
	Default interface{}   `json:"default"`
	Options []OptionValue `json:"options,omitempty"`
}

// OptionValue represents a value option for select-type game options
type OptionValue struct {
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Color options for color mode
var colorOptions = []string{"R", "B", "G", "Y", "O", "P"} // Red, Blue, Green, Yellow, Orange, Purple

// GenerateSecretCode creates a random code based on mode
// mode: "colors" - 4 unique colors from 6 options
// mode: "numbers" - 5 unique digits from 0-9
func GenerateSecretCode(mode string) string {
	rand.Seed(time.Now().UnixNano())
	code := ""
	used := make(map[string]bool)

	if mode == "colors" {
		// Generate 4 random unique colors (4 colors from 6 options)
		for len(code) < 4 {
			color := colorOptions[rand.Intn(len(colorOptions))]
			if !used[color] {
				code += color
				used[color] = true
			}
		}
	} else {
		// Generate 5 random unique digits (5 digits from 0-9)
		numberOptions := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
		for len(code) < 5 {
			num := numberOptions[rand.Intn(len(numberOptions))]
			if !used[num] {
				code += num
				used[num] = true
			}
		}
	}

	return code
}

// ValidateGuess checks if a guess is valid for the given mode
func ValidateGuess(guess, mode string) error {
	guess = strings.ToUpper(guess)

	// Check length based on mode
	if mode == "colors" {
		if len(guess) != 4 {
			return fmt.Errorf("guess must be exactly 4 characters for colors mode")
		}
	} else {
		if len(guess) != 5 {
			return fmt.Errorf("guess must be exactly 5 digits for numbers mode")
		}
	}

	// Check for duplicate characters
	seen := make(map[rune]bool)
	for _, char := range guess {
		if seen[char] {
			return fmt.Errorf("duplicate character not allowed: %c", char)
		}
		seen[char] = true
	}

	if mode == "colors" {
		// Each character must be a valid color
		for _, char := range guess {
			valid := false
			for _, color := range colorOptions {
				if string(char) == color {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid color: %c (must be R, B, G, Y, O, or P)", char)
			}
		}
	} else {
		// Each character must be a digit 0-9
		for _, char := range guess {
			if char < '0' || char > '9' {
				return fmt.Errorf("invalid digit: %c (must be 0-9)", char)
			}
		}
	}

	return nil
}

// CalculateBullsAndCows scores a guess against the secret code
// Bulls: correct color/number in correct position
// Cows: correct color/number in wrong position
func CalculateBullsAndCows(secret, guess string) (bulls, cows int) {
	secret = strings.ToUpper(secret)
	guess = strings.ToUpper(guess)

	// Track which positions are bulls
	secretUsed := make([]bool, len(secret))
	guessUsed := make([]bool, len(guess))

	// First pass: count bulls (exact matches)
	for i := 0; i < len(guess); i++ {
		if guess[i] == secret[i] {
			bulls++
			secretUsed[i] = true
			guessUsed[i] = true
		}
	}

	// Second pass: count cows (correct but wrong position)
	for i := 0; i < len(guess); i++ {
		if guessUsed[i] {
			continue // Skip bulls
		}
		for j := 0; j < len(secret); j++ {
			if secretUsed[j] {
				continue // Skip already matched
			}
			if guess[i] == secret[j] {
				cows++
				secretUsed[j] = true
				break
			}
		}
	}

	return bulls, cows
}

// CheckWin determines if the guess is a winning guess
// For colors: bulls == 4, for numbers: bulls == 5
func CheckWin(bulls int, mode string) bool {
	if mode == "colors" {
		return bulls == 4
	}
	return bulls == 5
}

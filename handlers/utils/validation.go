package utils

import (
	"regexp"
	"strings"
)

// IsValidEmail checks if an email address is valid
func IsValidEmail(email string) bool {
	// Use regex for more robust validation
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// IsValidUsername checks if a username is valid
func IsValidUsername(username string) bool {
	return len(strings.TrimSpace(username)) >= 3
} 
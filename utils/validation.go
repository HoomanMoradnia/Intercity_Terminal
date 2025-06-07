package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// IsValidEmail checks if the email format is valid.
func IsValidEmail(email string) bool {
	// A more robust regex for email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidUsername checks if the username format is valid based on specific criteria.
// It returns true if valid, and false with an error message if not.
func IsValidUsername(username string) (bool, string) {
	// 3-30 chars, Alphanumeric, underscore, hyphen
	if len(username) < 3 || len(username) > 30 {
		return false, "Username must be 3-30 characters long."
	}
	
	// Alphanumeric, underscore, hyphen
	if matched, _ := regexp.MatchString("^[0-9A-Za-z_-]+$", username); !matched {
		return false, "Username can only contain letters, numbers, underscores (_), or hyphens (-)."
	}

	// Cannot start with a number
	if unicode.IsDigit(rune(username[0])) {
		return false, "Username cannot start with a number."
	}

	// Cannot start or end with underscore or hyphen
	if strings.HasPrefix(username, "_") || strings.HasPrefix(username, "-") {
		return false, "Username cannot start with an underscore (_) or hyphen (-)."
	}
	if strings.HasSuffix(username, "_") || strings.HasSuffix(username, "-") {
		return false, "Username cannot end with an underscore (_) or hyphen (-)."
	}

	// Cannot contain consecutive underscores or hyphens
	if strings.Contains(username, "__") || strings.Contains(username, "--") || strings.Contains(username, "-_") || strings.Contains(username, "_-") {
		return false, "Username cannot contain consecutive underscores (_) or hyphens (-)."
	}

	return true, ""
}

// IsValidPassword checks if the password meets complexity requirements.
// It returns true if valid, and false with an error message if not.
func IsValidPassword(password string) (bool, string) {
	var (
		hasMinLen  = len(password) >= 8
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSymbol  = false
		errorMessages []string
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSymbol = true
		}
	}

	if !hasMinLen {
		errorMessages = append(errorMessages, "at least 8 characters")
	}
	if !hasUpper {
		errorMessages = append(errorMessages, "at least one uppercase letter")
	}
	if !hasLower {
		errorMessages = append(errorMessages, "at least one lowercase letter")
	}
	if !hasNumber {
		errorMessages = append(errorMessages, "at least one number")
	}
	if !hasSymbol {
		errorMessages = append(errorMessages, "at least one special character")
	}

	if len(errorMessages) > 0 {
		return false, fmt.Sprintf("Password must contain %s.", strings.Join(errorMessages, ", "))
	}

	return true, ""
} 
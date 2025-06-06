package tokens

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

// ResetTokenInfo contains info about a password reset token
type ResetTokenInfo struct {
	UserID int
	Expiry time.Time
}

var (
	resetTokens = make(map[string]ResetTokenInfo)
	tokenMutex  sync.RWMutex
)

const resetTokenValidity = 15 * time.Minute // Token valid for 15 minutes

// GenerateResetToken creates a secure random token.
func GenerateResetToken() (string, error) {
	b := make([]byte, 16) // 128 bits
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// StoreResetToken stores a token for a user.
func StoreResetToken(userID int) (string, error) {
	token, err := GenerateResetToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	resetTokens[token] = ResetTokenInfo{
		UserID: userID,
		Expiry: time.Now().Add(resetTokenValidity),
	}
	log.Printf("Stored reset token for user ID %d (expires %v)", userID, resetTokens[token].Expiry)
	return token, nil
}

// ValidateResetToken checks if a token is valid and returns the user ID.
func ValidateResetToken(token string) (int, bool) {
	tokenMutex.RLock()
	info, exists := resetTokens[token]
	valid := exists && !time.Now().After(info.Expiry)
	tokenMutex.RUnlock() // Release read lock early

	if exists && !valid { // Token expired, remove it
		tokenMutex.Lock()
		// Double-check it still exists and is expired after acquiring write lock
		infoCheck, existsCheck := resetTokens[token]
		if existsCheck && !time.Now().After(infoCheck.Expiry) {
			// Someone else might have validated/removed it between RUnlock and Lock
			// Or maybe it was updated/re-added? Treat as invalid but don't delete.
			valid = false // Ensure we return invalid
			log.Printf("Reset token %s state changed during validation attempt.", token)
		} else if existsCheck {
			delete(resetTokens, token)
			log.Printf("Reset token %s expired and removed.", token)
		} else {
			log.Printf("Reset token %s was already removed before delete.", token)
		}
		tokenMutex.Unlock()
	}

	if valid {
		log.Printf("Validated reset token %s for user ID %d.", token, info.UserID)
	} else {
		log.Printf("Reset token %s invalid or not found.", token)
	}
	return info.UserID, valid
}

// InvalidateResetToken removes a token from the store.
func InvalidateResetToken(token string) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	delete(resetTokens, token)
	log.Printf("Invalidated reset token %s.", token)
}

// Generate a random 6-digit code for verification
func GenerateVerificationCode() string {
	code := make([]byte, 3) // 3 bytes = 6 hex digits
	rand.Read(code)
	return fmt.Sprintf("%06x", code)[:6]
} 
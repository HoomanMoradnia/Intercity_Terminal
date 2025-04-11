package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// WARNING: Hardcoded key for demonstration ONLY. Use a secure key management system in production.
var aesKey []byte

func init() {
	// Generate a key for demo. Replace this with secure key loading.
	key := make([]byte, 32) // AES-256
	_, err := rand.Read(key)
	if err != nil {
		log.Fatalf("Failed to generate AES key: %v", err)
	}
	aesKey = key
	log.Println("WARNING: Using generated demo AES key. Store securely in production!")
}

// Encrypt encrypts plaintext using AES-GCM.
func Encrypt(plaintext string) (string, error) {
	if len(aesKey) == 0 {
		return "", errors.New("AES key is not initialized")
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-GCM.
func Decrypt(ciphertextHex string) (string, error) {
	if len(aesKey) == 0 {
		return "", errors.New("AES key is not initialized")
	}
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex ciphertext: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, encryptedMessage := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintextBytes, err := aesGCM.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		// Log decryption failures carefully - avoid leaking info
		log.Printf("Decryption failed (potential tampering or wrong key): %v", err)
		return "", errors.New("decryption failed") // Generic error to user
	}

	return string(plaintextBytes), nil
}

// HashPassword generates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with a stored bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

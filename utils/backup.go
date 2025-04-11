package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// BackupDatabase creates a backup of the SQLite database file
func BackupDatabase(dbPath string) (string, error) {
	if dbPath == "" {
		dbPath = filepath.Join("data", "securesignin.db")
	}

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "", fmt.Errorf("database file not found at %s", dbPath)
	}

	// Create backup directory if it doesn't exist
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Use a static backup filename (no timestamp)
	backupFilename := "securesignin.db.bak"
	backupPath := filepath.Join(backupDir, backupFilename)

	// Copy database file to backup location
	srcFile, err := os.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source database file: %w", err)
	}
	defer srcFile.Close()

	// Remove existing backup file if it exists
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return "", fmt.Errorf("failed to remove existing backup file: %w", err)
		}
	}

	destFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	// Perform the copy
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy database file: %w", err)
	}

	// Set file permissions to be readable/writable by owner, readable by others
	if err := os.Chmod(backupPath, 0644); err != nil {
		log.Printf("Warning: Failed to set backup file permissions: %v", err)
	}

	log.Printf("Successfully created database backup at %s (Last backup time: %s)",
		backupPath, time.Now().Format("2006-01-02 15:04:05"))
	return backupPath, nil
}

// ScheduleBackups sets up a ticker to perform regular database backups
func ScheduleBackups(dbPath string, intervalHours int) {
	if intervalHours <= 0 {
		intervalHours = 24 // Default to daily backups
	}

	// Set up a ticker to run at the specified interval
	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	go func() {
		for range ticker.C {
			backupPath, err := BackupDatabase(dbPath)
			if err != nil {
				log.Printf("Scheduled backup failed: %v", err)
			} else {
				log.Printf("Scheduled backup completed: %s", backupPath)
			}
		}
	}()

	log.Printf("Database backup scheduler started with %d hour interval", intervalHours)
}

// cleanupOldBackups is no longer needed since we only keep one backup file
// Keeping it for backward compatibility with any code that might call it
func cleanupOldBackups(backupDir string, keep int) {
	// This function is a no-op now since we maintain only one backup file
	log.Printf("Note: cleanupOldBackups is deprecated as the system now maintains a single backup file")
}

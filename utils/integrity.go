package utils

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// CheckDatabaseIntegrity verifies the integrity of the SQLite database
func CheckDatabaseIntegrity(dbPath string) (bool, string, error) {
	if dbPath == "" {
		dbPath = filepath.Join("data", "securesignin.db")
	}

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false, "", fmt.Errorf("database file not found at %s", dbPath)
	}

	// Open a new connection to the database specifically for integrity check
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to open database for integrity check: %w", err)
	}
	defer db.Close()

	// Run the integrity check
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		return false, "", fmt.Errorf("failed to run integrity check: %w", err)
	}
	defer rows.Close()

	var result string
	if rows.Next() {
		if err := rows.Scan(&result); err != nil {
			return false, "", fmt.Errorf("failed to scan integrity check result: %w", err)
		}
	}

	// Check the result (should be "ok" if database is healthy)
	isValid := result == "ok"
	log.Printf("Database integrity check result: %s", result)

	return isValid, result, nil
}

// RepairDatabaseIfNeeded attempts to repair a corrupted SQLite database
func RepairDatabaseIfNeeded(dbPath string) error {
	// First run an integrity check
	isValid, result, err := CheckDatabaseIntegrity(dbPath)
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	// If database is already valid, return success
	if isValid {
		log.Printf("Database is healthy, no repair needed")
		return nil
	}

	log.Printf("Database integrity check failed with result: %s", result)
	log.Printf("Attempting to repair database at %s", dbPath)

	// Create a backup before attempting repair
	backupPath, err := BackupDatabase(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create backup before repair: %w", err)
	}
	log.Printf("Created backup at %s before repair attempt", backupPath)

	// Attempt vacuum to rebuild the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database for repair: %w", err)
	}
	defer db.Close()

	// Try to run VACUUM to rebuild the database file
	if _, err := db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum operation failed: %w", err)
	}
	log.Printf("VACUUM operation completed successfully")

	// Check integrity again after repair
	isValid, result, err = CheckDatabaseIntegrity(dbPath)
	if err != nil {
		return fmt.Errorf("post-repair integrity check failed: %w", err)
	}

	if !isValid {
		return fmt.Errorf("repair attempt unsuccessful, database still corrupt: %s", result)
	}

	log.Printf("Database repair successful, integrity check passed")
	return nil
}

// VerifyTableConsistency checks if all required tables and indexes exist
func VerifyTableConsistency(db *sql.DB) error {
	// List of expected tables
	expectedTables := []string{"users", "login_history"}

	// Query to check if a table exists
	checkTableQuery := `
	SELECT name FROM sqlite_master 
	WHERE type='table' AND name=?
	`

	for _, tableName := range expectedTables {
		var name string
		err := db.QueryRow(checkTableQuery, tableName).Scan(&name)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("missing required table: %s", tableName)
			}
			return fmt.Errorf("error checking table %s: %w", tableName, err)
		}
	}

	// Check for username index
	var indexName string
	err := db.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='index' AND name='idx_username'
	`).Scan(&indexName)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("missing required index: idx_username")
		}
		return fmt.Errorf("error checking idx_username index: %w", err)
	}

	log.Printf("All required tables and indexes verified")
	return nil
}

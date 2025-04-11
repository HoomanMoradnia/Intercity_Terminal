package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SecureSignIn/utils"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

var DB *sql.DB

// InitializeDB sets up the SQLite database connection.
func InitializeDB() error {
	var err error

	// Check for a custom database path from environment variable
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		// Use default path in data directory
		dataDir := "data"
		// Ensure data directory exists
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
		dbPath = filepath.Join(dataDir, "securesignin.db")
	}

	log.Printf("Opening SQLite database at: %s", dbPath)
	// SQLite connection string with WAL journal mode for better concurrency and performance
	connStr := fmt.Sprintf("%s?_journal=WAL&_timeout=5000&_fk=true", dbPath)
	DB, err = sql.Open("sqlite3", connStr)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// SQLite doesn't need as many connections as PostgreSQL
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Check connection
	pingErr := DB.Ping()
	if pingErr != nil {
		DB.Close() // Ensure connection is closed if ping fails
		return fmt.Errorf("failed to connect to database: %w", pingErr)
	}

	// Apply SQLite performance optimizations
	pragmas := []string{
		"PRAGMA synchronous = NORMAL", // Faster, still safe for most cases
		"PRAGMA journal_mode = WAL",   // Enable Write-Ahead Logging
		"PRAGMA foreign_keys = ON",    // Enforce foreign key constraints
		"PRAGMA cache_size = 5000",    // Use more memory for caching (in pages)
		"PRAGMA temp_store = MEMORY",  // Store temporary tables in memory
	}

	for _, pragma := range pragmas {
		if _, err := DB.Exec(pragma); err != nil {
			log.Printf("Warning: Failed to execute pragma '%s': %v", pragma, err)
		} else {
			log.Printf("Applied SQLite optimization: %s", pragma)
		}
	}

	// Check if database file exists and has data
	fileInfo, err := os.Stat(dbPath)
	isNewDB := false
	if err != nil {
		if os.IsNotExist(err) {
			isNewDB = true
			log.Println("Database file does not exist, will be created")
		} else {
			log.Printf("Error checking database file: %v", err)
		}
	} else {
		isNewDB = fileInfo.Size() == 0
		if isNewDB {
			log.Println("Database file exists but is empty")
		}
	}

	// For existing databases, perform integrity check
	if !isNewDB {
		// Check database integrity
		isValid, result, err := utils.CheckDatabaseIntegrity(dbPath)
		if err != nil {
			log.Printf("Warning: Database integrity check failed: %v", err)
		} else if !isValid {
			log.Printf("Warning: Database integrity check returned: %s", result)
			log.Println("Attempting automatic database repair...")

			if err := utils.RepairDatabaseIfNeeded(dbPath); err != nil {
				log.Printf("Database repair failed: %v", err)
				log.Println("Continuing with potentially corrupted database - data loss may occur")
			} else {
				log.Println("Database repair completed successfully")
			}
		} else {
			log.Println("Database integrity check passed")
		}
	}

	log.Println("Database connection successful. Initializing schema...")
	err = initializeSchema()
	if err != nil {
		DB.Close()
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	// Verify that all expected tables and indexes exist
	if err := utils.VerifyTableConsistency(DB); err != nil {
		log.Printf("Warning: Table consistency check failed: %v", err)
		// We continue anyway as the schema initialization should have created any missing tables
	}

	log.Println("Database initialized successfully")
	return nil
}

// initializeSchema creates the necessary tables (SQLite syntax).
func initializeSchema() error {
	// Create users table with all essential columns
	usersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		date_of_birth TEXT,
		social_security TEXT
	);
	`
	_, err := DB.Exec(usersTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create login_history table if it doesn't exist
	loginHistoryTableSQL := `
	CREATE TABLE IF NOT EXISTS login_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		login_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT,
		success INTEGER,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err = DB.Exec(loginHistoryTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create login_history table: %w", err)
	}

	// Add unique index separately
	usernameIndexSQL := `CREATE UNIQUE INDEX IF NOT EXISTS idx_username ON users(username);`
	_, err = DB.Exec(usernameIndexSQL)
	if err != nil {
		log.Printf("Warning: Could not ensure username index (might already exist): %v", err)
	}

	return nil
}

// AddUser inserts a new user into the database (SQLite).
func AddUser(username, passwordHash, dob, ssn string) (int64, error) {
	query := "INSERT INTO users (username, password, date_of_birth, social_security) VALUES (?, ?, ?, ?)"
	result, err := DB.Exec(query, username, passwordHash, dob, ssn)
	if err != nil {
		return 0, fmt.Errorf("error inserting user (sqlite): %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert ID: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves a user by their username (SQLite).
func GetUserByUsername(username string) (*sql.Row, error) {
	query := "SELECT id, username, date_of_birth, social_security, password FROM users WHERE username = ?"
	row := DB.QueryRow(query, username)
	return row, nil // Error checking deferred to Scan
}

// GetUserByID retrieves a user by their ID (SQLite).
func GetUserByID(userID int) (*sql.Row, error) {
	query := "SELECT id, username, password FROM users WHERE id = ?"
	row := DB.QueryRow(query, userID)
	return row, nil // Error checking deferred to Scan
}

// UpdateUserPassword updates a user's password hash (SQLite).
func UpdateUserPassword(userID int, newPasswordHash string) error {
	query := "UPDATE users SET password = ? WHERE id = ?"
	_, err := DB.Exec(query, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("error updating password for user ID %d: %w", userID, err)
	}
	return nil
}

// LogLoginAttempt records a login attempt (SQLite).
func LogLoginAttempt(userID int64, ipAddress string, success bool) error {
	// Convert bool to integer for SQLite
	successInt := 0
	if success {
		successInt = 1
	}

	query := "INSERT INTO login_history (user_id, ip_address, success) VALUES (?, ?, ?)"
	_, err := DB.Exec(query, userID, ipAddress, successInt)
	if err != nil {
		return fmt.Errorf("error logging login attempt for user ID %d: %w", userID, err)
	}
	return nil
}

// GetAllUsers retrieves all users (SQLite).
func GetAllUsers() (*sql.Rows, error) {
	rows, err := DB.Query("SELECT id, username, created_at FROM users ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("error retrieving all users: %w", err)
	}
	return rows, nil
}

// GetLoginHistory retrieves all login history records (SQLite).
func GetLoginHistory() (*sql.Rows, error) {
	query := `
	SELECT lh.id, u.username, lh.login_time, lh.ip_address, lh.success
	FROM login_history lh
	JOIN users u ON lh.user_id = u.id
	ORDER BY lh.login_time DESC
	LIMIT 100
	`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error retrieving login history: %w", err)
	}
	return rows, nil
}

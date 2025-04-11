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
		email TEXT NOT NULL,
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

	// Create reset_codes table
	resetCodesTableSQL := `
	CREATE TABLE IF NOT EXISTS reset_codes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		code TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		used INTEGER DEFAULT 0,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err = DB.Exec(resetCodesTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create reset_codes table: %w", err)
	}

	// Create security_questions table
	securityQuestionsTableSQL := `
	CREATE TABLE IF NOT EXISTS security_questions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		question TEXT NOT NULL,
		answer_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err = DB.Exec(securityQuestionsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create security_questions table: %w", err)
	}

	// Add unique index separately
	usernameIndexSQL := `CREATE UNIQUE INDEX IF NOT EXISTS idx_username ON users(username);`
	_, err = DB.Exec(usernameIndexSQL)
	if err != nil {
		log.Printf("Warning: Could not ensure username index (might already exist): %v", err)
	}

	// Run migrations to update existing schema if needed
	err = migrateSchema()
	if err != nil {
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	return nil
}

// migrateSchema updates the database schema if it is outdated
func migrateSchema() error {
	log.Println("Checking if database schema needs migration...")

	// Check if email column exists in users table
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='email'`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if email column exists: %w", err)
	}

	if count == 0 {
		log.Println("Adding email column to users table...")
		// SQLite doesn't support adding UNIQUE constraint with ALTER TABLE,
		// so we just add the column without the constraint
		_, err := DB.Exec(`ALTER TABLE users ADD COLUMN email TEXT`)
		if err != nil {
			return fmt.Errorf("failed to add email column: %w", err)
		}
		log.Println("Email column added successfully")
	} else {
		log.Println("Email column already exists in users table")
	}

	// Check if security_questions table exists
	var tableExists int
	err = DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='security_questions'`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check if security_questions table exists: %w", err)
	}

	if tableExists == 0 {
		log.Println("Creating security_questions table...")
		securityQuestionsTableSQL := `
		CREATE TABLE IF NOT EXISTS security_questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			question TEXT NOT NULL,
			answer_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
		`
		_, err := DB.Exec(securityQuestionsTableSQL)
		if err != nil {
			return fmt.Errorf("failed to create security_questions table: %w", err)
		}
		log.Println("security_questions table created successfully")
	} else {
		log.Println("security_questions table already exists")
	}

	return nil
}

// AddUser inserts a new user into the database (SQLite).
func AddUser(username, passwordHash, dob, ssn string, email string) (int64, error) {
	// Validate required fields
	if username == "" || passwordHash == "" || email == "" {
		return 0, fmt.Errorf("username, password, and email are required fields")
	}

	// Check if email column exists
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='email'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to check if email column exists: %w", err)
	}

	var result sql.Result
	if count > 0 {
		// Table has email column
		query := "INSERT INTO users (username, password, date_of_birth, social_security, email) VALUES (?, ?, ?, ?, ?)"
		result, err = DB.Exec(query, username, passwordHash, dob, ssn, email)
	} else {
		// Fall back to original schema without email
		query := "INSERT INTO users (username, password, date_of_birth, social_security) VALUES (?, ?, ?, ?)"
		result, err = DB.Exec(query, username, passwordHash, dob, ssn)
		log.Printf("Warning: User created without email because email column doesn't exist")
	}

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
	// Check if email column exists
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='email'`).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check if email column exists: %w", err)
	}

	var row *sql.Row
	if count > 0 {
		// Table has email column
		query := "SELECT id, username, date_of_birth, social_security, password, email FROM users WHERE username = ?"
		row = DB.QueryRow(query, username)
	} else {
		// Fall back to original schema without email
		query := "SELECT id, username, date_of_birth, social_security, password, '' as email FROM users WHERE username = ?"
		row = DB.QueryRow(query, username)
	}

	return row, nil // Error checking deferred to Scan
}

// GetUserByEmail retrieves a user by their email.
func GetUserByEmail(email string) (*sql.Row, error) {
	// Check if email column exists
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='email'`).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check if email column exists: %w", err)
	}

	if count == 0 {
		// Email column doesn't exist
		return nil, fmt.Errorf("email column does not exist in users table")
	}

	// Return the same structure as GetUserByUsername for consistency
	query := "SELECT id, username, date_of_birth, social_security, password, email FROM users WHERE email = ?"
	row := DB.QueryRow(query, email)
	return row, nil
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

// StoreResetCode stores a new reset code for a user
func StoreResetCode(userID int64, code string, expiresAt time.Time) error {
	query := "INSERT INTO reset_codes (user_id, code, expires_at) VALUES (?, ?, ?)"
	_, err := DB.Exec(query, userID, code, expiresAt)
	if err != nil {
		return fmt.Errorf("error storing reset code: %w", err)
	}
	return nil
}

// ValidateResetCode checks if a reset code is valid and not expired
func ValidateResetCode(code string) (int64, error) {
	query := `
		SELECT user_id 
		FROM reset_codes 
		WHERE code = ? 
		AND used = 0 
		AND expires_at > CURRENT_TIMESTAMP
		LIMIT 1
	`
	var userID int64
	err := DB.QueryRow(query, code).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("invalid or expired reset code")
		}
		return 0, fmt.Errorf("error validating reset code: %w", err)
	}
	return userID, nil
}

// MarkResetCodeUsed marks a reset code as used
func MarkResetCodeUsed(code string) error {
	query := "UPDATE reset_codes SET used = 1 WHERE code = ?"
	_, err := DB.Exec(query, code)
	if err != nil {
		return fmt.Errorf("error marking reset code as used: %w", err)
	}
	return nil
}

// CheckEmailExists checks if an email address already exists in the database
func CheckEmailExists(email string) (bool, error) {
	// Check if email column exists
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='email'`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if email column exists: %w", err)
	}

	if count == 0 {
		// Email column doesn't exist, so no email can exist
		return false, nil
	}

	// Check if email exists
	var exists int
	query := "SELECT COUNT(*) FROM users WHERE email = ?"
	err = DB.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking if email exists: %w", err)
	}

	return exists > 0, nil
}

// --- Security Questions Functions ---

// AddSecurityQuestion adds a security question and answer for a user
func AddSecurityQuestion(userID int64, question string, answerHash string) error {
	query := "INSERT INTO security_questions (user_id, question, answer_hash) VALUES (?, ?, ?)"
	_, err := DB.Exec(query, userID, question, answerHash)
	if err != nil {
		return fmt.Errorf("error adding security question for user ID %d: %w", userID, err)
	}
	return nil
}

// GetSecurityQuestionByUserID retrieves the security question for a user
func GetSecurityQuestionByUserID(userID int64) (int64, string, string, error) {
	query := "SELECT id, question, answer_hash FROM security_questions WHERE user_id = ? LIMIT 1"
	var id int64
	var question, answerHash string
	err := DB.QueryRow(query, userID).Scan(&id, &question, &answerHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", "", fmt.Errorf("no security question found for user ID %d", userID)
		}
		return 0, "", "", fmt.Errorf("error getting security question for user ID %d: %w", userID, err)
	}
	return id, question, answerHash, nil
}

// UpdateSecurityQuestion updates a user's security question and answer
func UpdateSecurityQuestion(questionID int64, question string, answerHash string) error {
	query := "UPDATE security_questions SET question = ?, answer_hash = ? WHERE id = ?"
	_, err := DB.Exec(query, question, answerHash, questionID)
	if err != nil {
		return fmt.Errorf("error updating security question ID %d: %w", questionID, err)
	}
	return nil
}

// HasSecurityQuestion checks if a user has set up a security question
func HasSecurityQuestion(userID int64) (bool, error) {
	query := "SELECT COUNT(*) FROM security_questions WHERE user_id = ?"
	var count int
	err := DB.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking if user ID %d has security questions: %w", userID, err)
	}
	return count > 0, nil
}

// DeleteSecurityQuestions removes all security questions for a user
func DeleteSecurityQuestions(userID int64) error {
	query := "DELETE FROM security_questions WHERE user_id = ?"
	_, err := DB.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("error deleting security questions for user ID %d: %w", userID, err)
	}
	return nil
}

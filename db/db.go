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
	"golang.org/x/crypto/bcrypt"
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

	// Apply any schema migrations
	if err = migrateSchema(); err != nil {
		log.Printf("Warning: Schema migration failed: %v", err)
		// Continue anyway, as the application might still work with the existing schema
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
		social_security TEXT,
		role TEXT DEFAULT 'Operator'
	);
	`
	_, err := DB.Exec(usersTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create vehicles table
	vehiclesTableSQL := `
	CREATE TABLE IF NOT EXISTS vehicles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vehicle_number TEXT UNIQUE NOT NULL,
		type TEXT NOT NULL,
		capacity INTEGER NOT NULL,
		status TEXT NOT NULL,
		last_maintenance_date TEXT,
		next_maintenance_date TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		notes TEXT
	);
	`
	_, err = DB.Exec(vehiclesTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create vehicles table: %w", err)
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

	// Create trips table
	tripsTableSQL := `
	CREATE TABLE IF NOT EXISTS trips (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		origin TEXT NOT NULL,
		destination TEXT NOT NULL,
		vehicle_id INTEGER NOT NULL,
		departure_time TEXT NOT NULL,
		arrival_time TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(vehicle_id) REFERENCES vehicles(id) ON DELETE SET NULL
	);
	`
	_, err = DB.Exec(tripsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create trips table: %w", err)
	}

	// Create bookings table
	bookingsTableSQL := `
	CREATE TABLE IF NOT EXISTS bookings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trip_id INTEGER NOT NULL,
		passenger TEXT NOT NULL,
		social_id TEXT NOT NULL,
		phone_number TEXT NOT NULL,
		date_of_birth TEXT NOT NULL,
		booking_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT NOT NULL,
		FOREIGN KEY(trip_id) REFERENCES trips(id) ON DELETE CASCADE
	);
	`
	_, err = DB.Exec(bookingsTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create bookings table: %w", err)
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

	// Check if vehicles table exists
	var vehiclesTableExists int
	err = DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='vehicles'`).Scan(&vehiclesTableExists)
	if err != nil {
		return fmt.Errorf("failed to check if vehicles table exists: %w", err)
	}

	if vehiclesTableExists == 0 {
		log.Println("Creating vehicles table...")
		vehiclesTableSQL := `
		CREATE TABLE IF NOT EXISTS vehicles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vehicle_number TEXT UNIQUE NOT NULL,
			type TEXT NOT NULL,
			capacity INTEGER NOT NULL,
			status TEXT NOT NULL,
			last_maintenance_date TEXT,
			next_maintenance_date TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			notes TEXT
		);
		`
		_, err := DB.Exec(vehiclesTableSQL)
		if err != nil {
			return fmt.Errorf("failed to create vehicles table: %w", err)
		}
		log.Println("vehicles table created successfully")
	} else {
		log.Println("vehicles table already exists")
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

	// Check if the role column exists in the users table
	var roleExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='role'").Scan(&roleExists)
	if err != nil {
		return fmt.Errorf("failed to check if role column exists: %w", err)
	}

	// If role column does not exist, add it
	if roleExists == 0 {
		log.Println("Adding 'role' column to users table...")
		_, err = DB.Exec("ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'Operator'")
		if err != nil {
			return fmt.Errorf("failed to add role column: %w", err)
		}
		log.Println("Added 'role' column to users table")
	}

	return nil
}

// AddUser creates a new user record in the database
func AddUser(username, passwordHash, dob, ssn string, email string, role string) (int64, error) {
	// Validate inputs (basic check)
	if username == "" || passwordHash == "" || email == "" {
		return 0, fmt.Errorf("username, password, and email are required")
	}

	// If role is empty, default to Operator
	if role == "" {
		role = "Operator"
	}

	// Prepare the SQL statement for inserting a new user
	stmt, err := DB.Prepare(`
		INSERT INTO users (username, password, date_of_birth, social_security, email, role)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute the statement with the provided values
	result, err := stmt.Exec(username, passwordHash, dob, ssn, email, role)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}

	// Get the ID of the inserted user
	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return userID, nil
}

// GetUserByUsername retrieves a user by username.
func GetUserByUsername(username string) (*sql.Row, error) {
	// First check if the username exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("error checking if username exists: %w", err)
	}

	if count == 0 {
		// Return empty row, which will result in sql.ErrNoRows when scanned
		return DB.QueryRow("SELECT 1 WHERE 1=0"), nil
	}

	// Get user details including role
	query := `
		SELECT id, username, date_of_birth, social_security, password, email, COALESCE(role, 'Operator') as role
		FROM users
		WHERE username = ?
	`
	row := DB.QueryRow(query, username)
	return row, nil
}

// GetUserByEmail retrieves a user by email.
func GetUserByEmail(email string) (*sql.Row, error) {
	// First check if the email exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("error checking if email exists: %w", err)
	}

	if count == 0 {
		// Return empty row, which will result in sql.ErrNoRows when scanned
		return DB.QueryRow("SELECT 1 WHERE 1=0"), nil
	}

	// Get user details including role
	query := `
		SELECT id, username, date_of_birth, social_security, password, email, COALESCE(role, 'Operator') as role
		FROM users
		WHERE email = ?
	`
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

// UpdateUsername updates a user's username in the database.
func UpdateUsername(userID int64, newUsername string) error {
	// Check if username already exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? AND id != ?", newUsername, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("error checking if username exists: %w", err)
	}
	
	if count > 0 {
		return fmt.Errorf("username '%s' is already taken", newUsername)
	}
	
	// Update the username
	query := "UPDATE users SET username = ? WHERE id = ?"
	_, err = DB.Exec(query, newUsername, userID)
	if err != nil {
		return fmt.Errorf("error updating username for user ID %d: %w", userID, err)
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
	rows, err := DB.Query("SELECT id, username, email, password, created_at, date_of_birth, social_security, role FROM users ORDER BY username")
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

// EnsureAdminUser creates a default admin user if no admin exists
func EnsureAdminUser() error {
	// Check if an admin user already exists
	query := "SELECT COUNT(*) FROM users WHERE role = 'Admin'"
	var count int
	err := DB.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("error checking for admin users: %w", err)
	}

	// If no admin users exist, create one
	if count == 0 {
		log.Println("No admin users found. Creating default admin user...")
		
		// Hash the default password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("error hashing default admin password: %w", err)
		}
		
		// Insert the admin user
		_, err = AddUser("admin", string(hashedPassword), "", "", "admin@example.com", "Admin")
		if err != nil {
			return fmt.Errorf("error creating default admin user: %w", err)
		}
		
		log.Println("Default admin user created successfully. Username: admin, Password: admin")
	}
	
	return nil
}

// UpdateUserRole updates a user's role in the database.
func UpdateUserRole(userID int64, newRole string) error {
	query := "UPDATE users SET role = ? WHERE id = ?"
	_, err := DB.Exec(query, newRole, userID)
	if err != nil {
		return fmt.Errorf("error updating role for user ID %d: %w", userID, err)
	}
	return nil
}

// DeleteUser deletes a user from the database.
func DeleteUser(userID int64) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := DB.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("error deleting user ID %d: %w", userID, err)
	}
	return nil
}

// --- Vehicle Functions ---

// AddVehicle adds a new vehicle to the database
func AddVehicle(vehicleNumber, vehicleType string, capacity int, status, lastMaintenance, nextMaintenance, notes string) (int64, error) {
	// Validate inputs (basic check)
	if vehicleNumber == "" || vehicleType == "" || status == "" {
		return 0, fmt.Errorf("vehicle number, type, and status are required")
	}

	// Prepare the SQL statement for inserting a new vehicle
	stmt, err := DB.Prepare(`
		INSERT INTO vehicles (vehicle_number, type, capacity, status, last_maintenance_date, next_maintenance_date, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute the statement with the provided values
	result, err := stmt.Exec(vehicleNumber, vehicleType, capacity, status, lastMaintenance, nextMaintenance, notes)
	if err != nil {
		return 0, fmt.Errorf("failed to insert vehicle: %w", err)
	}

	// Get the ID of the inserted vehicle
	vehicleID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return vehicleID, nil
}

// GetAllVehicles retrieves all vehicles from the database
func GetAllVehicles() (*sql.Rows, error) {
	rows, err := DB.Query(`
		SELECT id, vehicle_number, type, capacity, status, 
			   last_maintenance_date, next_maintenance_date, created_at, notes 
		FROM vehicles 
		ORDER BY vehicle_number
	`)
	if err != nil {
		return nil, fmt.Errorf("error retrieving all vehicles: %w", err)
	}
	return rows, nil
}

// GetAvailableVehicles retrieves vehicles that are not under repair, not scheduled for maintenance during the given time range, and not assigned to overlapping trips
func GetAvailableVehicles(departureTime, arrivalTime string) (*sql.Rows, error) {
	query := `
		SELECT id, vehicle_number, type, capacity, status,
		       last_maintenance_date, next_maintenance_date, created_at, notes
		FROM vehicles
		WHERE status != 'Under repair'
		  AND (next_maintenance_date IS NULL OR next_maintenance_date < ? OR next_maintenance_date > ?)
		  AND id NOT IN (
			SELECT vehicle_id FROM trips
			WHERE departure_time < ? AND arrival_time > ?
		  )
		ORDER BY vehicle_number
	`
	// Parameter order: maintenance before, maintenance after, new_arrival, new_departure
	rows, err := DB.Query(query, departureTime, arrivalTime, arrivalTime, departureTime)
	if err != nil {
		return nil, fmt.Errorf("error retrieving available vehicles: %w", err)
	}
	return rows, nil
}

// GetVehicleByID retrieves a vehicle by its ID
func GetVehicleByID(vehicleID int64) (*sql.Row, error) {
	query := `
		SELECT id, vehicle_number, type, capacity, status, 
			   last_maintenance_date, next_maintenance_date, created_at, notes 
		FROM vehicles 
		WHERE id = ?
	`
	row := DB.QueryRow(query, vehicleID)
	return row, nil
}

// UpdateVehicle updates a vehicle's information in the database
func UpdateVehicle(vehicleID int64, vehicleNumber, vehicleType string, capacity int, 
				  status, lastMaintenance, nextMaintenance, notes string) error {
	// Validate inputs
	if vehicleNumber == "" || vehicleType == "" || status == "" {
		return fmt.Errorf("vehicle number, type, and status are required")
	}

	// Check if vehicle number already exists for a different vehicle
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM vehicles WHERE vehicle_number = ? AND id != ?", 
					  vehicleNumber, vehicleID).Scan(&count)
	if err != nil {
		return fmt.Errorf("error checking if vehicle number exists: %w", err)
	}
	
	if count > 0 {
		return fmt.Errorf("vehicle number '%s' is already in use", vehicleNumber)
	}

	query := `
		UPDATE vehicles 
		SET vehicle_number = ?, 
			type = ?, 
			capacity = ?, 
			status = ?, 
			last_maintenance_date = ?, 
			next_maintenance_date = ?, 
			notes = ? 
		WHERE id = ?
	`
	_, err = DB.Exec(query, vehicleNumber, vehicleType, capacity, status, 
					lastMaintenance, nextMaintenance, notes, vehicleID)
	if err != nil {
		return fmt.Errorf("error updating vehicle with ID %d: %w", vehicleID, err)
	}
	return nil
}

// DeleteVehicle deletes a vehicle from the database
func DeleteVehicle(vehicleID int64) error {
	query := "DELETE FROM vehicles WHERE id = ?"
	_, err := DB.Exec(query, vehicleID)
	if err != nil {
		return fmt.Errorf("error deleting vehicle with ID %d: %w", vehicleID, err)
	}
	return nil
}

// --- Trip Functions ---

// AddTrip adds a new trip to the database
func AddTrip(origin, destination string, vehicleID int64, departureTime, arrivalTime string) (int64, error) {
	// Validate departure before arrival
	if departureTime >= arrivalTime {
		return 0, fmt.Errorf("departure time must be before arrival time")
	}
	
	// Validate origin and destination are different
	if origin == destination {
		return 0, fmt.Errorf("origin and destination cannot be the same city")
	}
	
	stmt, err := DB.Prepare(`
		INSERT INTO trips (origin, destination, vehicle_id, departure_time, arrival_time)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare trip insert: %w", err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(origin, destination, vehicleID, departureTime, arrivalTime)
	if err != nil {
		return 0, fmt.Errorf("failed to insert trip: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve trip id: %w", err)
	}
	return id, nil
}

// GetAllTrips retrieves all trips
func GetAllTrips() (*sql.Rows, error) {
	rows, err := DB.Query(`
		SELECT t.id, t.origin, t.destination, t.vehicle_id, t.departure_time, t.arrival_time, t.created_at, v.vehicle_number
		FROM trips t
		LEFT JOIN vehicles v ON t.vehicle_id = v.id
		ORDER BY t.departure_time DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error retrieving trips: %w", err)
	}
	return rows, nil
}

// UpdateTrip updates trip details
func UpdateTrip(id int64, origin, destination string, vehicleID int64, departureTime, arrivalTime string) error {
	// Validate departure before arrival
	if departureTime >= arrivalTime {
		return fmt.Errorf("departure time must be before arrival time")
	}
	
	// Validate origin and destination are different
	if origin == destination {
		return fmt.Errorf("origin and destination cannot be the same city")
	}
	
	stmt, err := DB.Prepare(`
		UPDATE trips SET origin=?, destination=?, vehicle_id=?, departure_time=?, arrival_time=? WHERE id=?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare trip update: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(origin, destination, vehicleID, departureTime, arrivalTime, id)
	if err != nil {
		return fmt.Errorf("failed to update trip: %w", err)
	}
	return nil
}

// DeleteTrip deletes a trip by ID
func DeleteTrip(id int64) error {
	_, err := DB.Exec("DELETE FROM trips WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("failed to delete trip: %w", err)
	}
	return nil
}

// AddBooking adds a new booking to the database
func AddBooking(tripID int64, passenger, socialID, phoneNumber, dateOfBirth, status string) (int64, error) {
	// Validate inputs
	if tripID == 0 || passenger == "" || status == "" {
		return 0, fmt.Errorf("trip ID, passenger name, and status are required")
	}
	
	// Prepare insert
	stmt, err := DB.Prepare("INSERT INTO bookings (trip_id, passenger, social_id, phone_number, date_of_birth, status) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare booking insert: %w", err)
	}
	defer stmt.Close()

	// Execute insert
	res, err := stmt.Exec(tripID, passenger, socialID, phoneNumber, dateOfBirth, status)
	if err != nil {
		return 0, fmt.Errorf("failed to insert booking: %w", err)
	}

	// Get new ID
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve booking id: %w", err)
	}
	return id, nil
}

// GetAllBookings retrieves all bookings
func GetAllBookings() (*sql.Rows, error) {
	return GetFilteredBookings(nil, "", "")
}

// UpdateBookingStatus updates the status of a booking
func UpdateBookingStatus(bookingID int64, status string) error {
	_, err := DB.Exec("UPDATE bookings SET status = ? WHERE id = ?", status, bookingID)
	if err != nil {
		return fmt.Errorf("error updating booking status: %w", err)
	}
	return nil
}

// DeleteBooking deletes a booking
func DeleteBooking(bookingID int64) error {
	_, err := DB.Exec("DELETE FROM bookings WHERE id = ?", bookingID)
	if err != nil {
		return fmt.Errorf("error deleting booking: %w", err)
	}
	return nil
}

// FindTripByRoute finds a trip by origin and destination
func FindTripByRoute(origin, destination string) (int64, error) {
	var tripID int64
	err := DB.QueryRow("SELECT id FROM trips WHERE origin = ? AND destination = ? ORDER BY departure_time LIMIT 1", origin, destination).Scan(&tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no trip found for route %s to %s", origin, destination)
		}
		return 0, fmt.Errorf("error finding trip: %w", err)
	}
	return tripID, nil
}

// GetFilteredBookings retrieves bookings with optional filtering and ordering
func GetFilteredBookings(filter map[string]string, orderBy string, orderDir string) (*sql.Rows, error) {
	query := `SELECT b.id, b.trip_id, b.passenger, b.social_id, b.phone_number, b.date_of_birth, b.booking_date, b.status, t.origin, t.destination, t.departure_time
	FROM bookings b
	JOIN trips t ON b.trip_id = t.id
	WHERE 1=1`
	
	var args []interface{}
	
	// Add filters
	if filter != nil {
		if v, ok := filter["passenger"]; ok && v != "" {
			query += " AND b.passenger LIKE ?"
			args = append(args, "%"+v+"%")
		}
		if v, ok := filter["origin"]; ok && v != "" {
			query += " AND t.origin = ?"
			args = append(args, v)
		}
		if v, ok := filter["destination"]; ok && v != "" {
			query += " AND t.destination = ?"
			args = append(args, v)
		}
		if v, ok := filter["status"]; ok && v != "" {
			query += " AND b.status = ?"
			args = append(args, v)
		}
		// Date range filters
		if v, ok := filter["date_from"]; ok && v != "" {
			query += " AND DATE(b.booking_date) >= ?"
			args = append(args, v)
		}
		if v, ok := filter["date_to"]; ok && v != "" {
			query += " AND DATE(b.booking_date) <= ?"
			args = append(args, v)
		}
	}
	
	// Add ordering
	if orderBy != "" {
		validColumns := map[string]string{
			"passenger": "b.passenger",
			"date": "b.booking_date",
			"status": "b.status",
			"origin": "t.origin",
			"destination": "t.destination",
		}
		
		if col, ok := validColumns[orderBy]; ok {
			query += " ORDER BY " + col
			if orderDir == "desc" {
				query += " DESC"
			} else {
				query += " ASC"
			}
		} else {
			query += " ORDER BY b.booking_date DESC"
		}
	} else {
		query += " ORDER BY b.booking_date DESC"
	}
	
	// Execute query
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error retrieving bookings: %w", err)
	}
	return rows, nil
}

// IsVehicleAvailableForTripEdit checks whether a vehicle is free for a given time range excluding a specific trip
func IsVehicleAvailableForTripEdit(vehicleID int64, departureTime, arrivalTime string, tripID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM vehicles v
		WHERE v.id = ?
		  AND v.status != 'Under repair'
		  AND (v.next_maintenance_date IS NULL OR v.next_maintenance_date < ? OR v.next_maintenance_date > ?)
		  AND NOT EXISTS (
			SELECT 1 FROM trips t
			WHERE t.vehicle_id = v.id
			  AND t.departure_time < ?
			  AND t.arrival_time > ?
			  AND t.id != ?
		  )`
	// Parameter order: vehicle, maintenance before, maintenance after, new_arrival, new_departure, exclude trip
	err := DB.QueryRow(query, vehicleID, departureTime, arrivalTime, arrivalTime, departureTime, tripID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking vehicle availability: %w", err)
	}
	return count > 0, nil
}

// GetTripVehicleCapacity retrieves the capacity of a vehicle assigned to a trip
func GetTripVehicleCapacity(tripID int64) (int, error) {
	var capacity int
	err := DB.QueryRow(`
		SELECT v.capacity 
		FROM trips t
		JOIN vehicles v ON t.vehicle_id = v.id
		WHERE t.id = ?
	`, tripID).Scan(&capacity)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("trip not found")
		}
		return 0, fmt.Errorf("error getting trip vehicle capacity: %w", err)
	}
	
	return capacity, nil
}

// GetTripBookingsCount retrieves the number of active bookings for a trip
func GetTripBookingsCount(tripID int64) (int, error) {
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*) 
		FROM bookings 
		WHERE trip_id = ? AND status != 'Cancelled'
	`, tripID).Scan(&count)
	
	if err != nil {
		return 0, fmt.Errorf("error getting trip bookings count: %w", err)
	}
	
	return count, nil
}

// GetVehicleBookingsCount retrieves the number of active bookings for all trips using a specific vehicle
func GetVehicleBookingsCount(vehicleID int64) (int, error) {
	var count int
	err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM bookings b
		JOIN trips t ON b.trip_id = t.id
		WHERE t.vehicle_id = ? AND b.status != 'Cancelled'
	`, vehicleID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error getting vehicle bookings count: %w", err)
	}
	return count, nil
}

// CheckTripAvailability verifies if a trip has available seats
func CheckTripAvailability(tripID int64) (bool, error) {
	capacity, err := GetTripVehicleCapacity(tripID)
	if err != nil {
		return false, err
	}
	
	bookingsCount, err := GetTripBookingsCount(tripID)
	if err != nil {
		return false, err
	}
	
	return bookingsCount < capacity, nil
}

// GetBookingSummary returns the count of bookings per day within a date range
func GetBookingSummary(from, to string) ([]map[string]interface{}, error) {
	query := `
	SELECT DATE(booking_date) AS date, COUNT(*) AS bookings
	FROM bookings
	WHERE DATE(booking_date) >= ? AND DATE(booking_date) <= ?
	GROUP BY DATE(booking_date)
	ORDER BY DATE(booking_date)
	`
	rows, err := DB.Query(query, from, to)
	if err != nil {
		return nil, fmt.Errorf("error getting booking summary: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			log.Printf("Error scanning booking summary row: %v", err)
			continue
		}
		results = append(results, map[string]interface{}{"date": date, "bookings": count})
	}
	return results, nil
}

// GetCancellationSummary returns daily bookings, cancellations, and cancellation rate within a date range
func GetCancellationSummary(from, to string) ([]map[string]interface{}, error) {
	query := `
	SELECT DATE(booking_date) AS date,
		   COUNT(*) AS bookings,
		   SUM(CASE WHEN status = 'Cancelled' THEN 1 ELSE 0 END) AS cancellations,
		   ROUND(
			 SUM(CASE WHEN status = 'Cancelled' THEN 1 ELSE 0 END) * 100.0 / COUNT(*),
			 2
		   ) AS cancellation_rate
	  FROM bookings
	 WHERE DATE(booking_date) >= ?
	   AND DATE(booking_date) <= ?
	 GROUP BY DATE(booking_date)
	 ORDER BY DATE(booking_date)
	`
	rows, err := DB.Query(query, from, to)
	if err != nil {
		return nil, fmt.Errorf("error getting cancellation summary: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var date string
		var total, cancels int
		var rate float64
		if err := rows.Scan(&date, &total, &cancels, &rate); err != nil {
			log.Printf("Error scanning cancellation summary row: %v", err)
			continue
		}
		results = append(results, map[string]interface{}{
			"date":               date,
			"bookings":           total,
			"cancellations":      cancels,
			"cancellation_rate":  rate,
		})
	}
	return results, nil
}

// GetRoutePerformance returns booking counts per origin-destination within a date range
func GetRoutePerformance(from, to string) ([]map[string]interface{}, error) {
	query := `
	SELECT t.origin, t.destination, COUNT(b.id) AS bookings
	FROM bookings b
	JOIN trips t ON b.trip_id = t.id
	WHERE DATE(b.booking_date) >= ? AND DATE(b.booking_date) <= ?
	GROUP BY t.origin, t.destination
	ORDER BY bookings DESC
	`
	rows, err := DB.Query(query, from, to)
	if err != nil {
		return nil, fmt.Errorf("error getting route performance: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var origin, destination string
		var count int
		if err := rows.Scan(&origin, &destination, &count); err != nil {
			log.Printf("Error scanning route performance row: %v", err)
			continue
		}
		results = append(results, map[string]interface{}{"origin": origin, "destination": destination, "bookings": count})
	}
	return results, nil
}

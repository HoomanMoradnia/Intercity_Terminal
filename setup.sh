#!/bin/bash

# Setup script for SecureSignIn with SQLite
# This script creates necessary directories and builds utility tools

# Create necessary directories
echo "Creating directory structure..."
mkdir -p data data/backups cmd/dbcheck cmd/dbbackup

# Create placeholder files if they don't exist
if [ ! -f cmd/dbcheck/main.go ]; then
    echo "Creating database check tool file..."
    cat > cmd/dbcheck/main.go << 'EOL'
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"SecureSignIn/utils"
)

func main() {
	// Define command-line flags
	var (
		dbPath     = flag.String("db", "", "Path to SQLite database file (default: data/securesignin.db)")
		repair     = flag.Bool("repair", false, "Attempt to repair database if integrity check fails")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
	)

	// Parse flags
	flag.Parse()

	// Get command (check, repair, etc.)
	cmd := "check"
	args := flag.Args()
	if len(args) > 0 {
		cmd = args[0]
	}

	// Set default database path if not provided
	if *dbPath == "" {
		*dbPath = filepath.Join("data", "securesignin.db")
	}

	// Execute command based on user input
	switch cmd {
	case "check":
		// Run integrity check
		if *verbose {
			log.Printf("Checking database integrity of %s...", *dbPath)
		}
		
		isValid, result, err := utils.CheckDatabaseIntegrity(*dbPath)
		if err != nil {
			log.Fatalf("Integrity check failed: %v", err)
		}
		
		if isValid {
			fmt.Println("✅ Database integrity check passed: database is OK")
		} else {
			fmt.Printf("❌ Database integrity check failed: %s\n", result)
		}
		
	default:
		fmt.Println("Unknown command. Available commands:")
		fmt.Println("  check  - Check database integrity")
	}
}
EOL
fi

if [ ! -f cmd/dbbackup/main.go ]; then
    echo "Creating database backup tool file..."
    cat > cmd/dbbackup/main.go << 'EOL'
package main

import (
	"fmt"
	"log"

	"SecureSignIn/utils"
)

func main() {
	backupPath, err := utils.BackupDatabase("")
	if err != nil {
		log.Fatalf("Backup failed: %v", err)
	}
	
	fmt.Printf("✅ Database backup created at %s\n", backupPath)
}
EOL
fi

# Check if go.mod needs SQLite driver
if ! grep -q "github.com/mattn/go-sqlite3" go.mod 2>/dev/null; then
    echo "Adding SQLite driver to go.mod..."
    go get github.com/mattn/go-sqlite3
fi

# Build the application
echo "Building application..."
go build -o securesignin .

# Set proper permissions
chmod 755 securesignin

echo "Setup completed successfully!"
echo "You can now run the application with: ./securesignin"
echo "Or use docker-compose: docker compose up -d"
echo ""
echo "Utility tools available:"
echo "- Database check: go run cmd/dbcheck/main.go check"
echo "- Database backup: go run cmd/dbbackup/main.go"
echo ""
echo "For data migration from PostgreSQL (if needed):"
echo "1. Ensure PostgreSQL is running"
echo "2. Run: go run migrate_postgres_to_sqlite.go" 
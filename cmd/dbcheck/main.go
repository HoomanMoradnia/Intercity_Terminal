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
		dbPath  = flag.String("db", "", "Path to SQLite database file (default: data/securesignin.db)")
		repair  = flag.Bool("repair", false, "Attempt to repair database if integrity check fails")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
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

	// Check if database file exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("Database file not found at %s", *dbPath)
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

			// Automatically repair if requested
			if *repair {
				fmt.Println("Attempting to repair database...")
				if err := utils.RepairDatabaseIfNeeded(*dbPath); err != nil {
					log.Fatalf("Database repair failed: %v", err)
				}
				fmt.Println("✅ Database repair completed successfully")
			} else {
				fmt.Println("Run with --repair flag to attempt automatic repair")
			}
		}

	case "repair":
		// Force repair
		if *verbose {
			log.Printf("Attempting to repair database %s...", *dbPath)
		}

		if err := utils.RepairDatabaseIfNeeded(*dbPath); err != nil {
			log.Fatalf("Database repair failed: %v", err)
		}

		fmt.Println("✅ Database repair completed successfully")

	case "backup":
		// Create backup
		if *verbose {
			log.Printf("Creating backup of database %s...", *dbPath)
		}

		backupPath, err := utils.BackupDatabase(*dbPath)
		if err != nil {
			log.Fatalf("Backup failed: %v", err)
		}

		fmt.Printf("✅ Database backup created at %s\n", backupPath)

	default:
		fmt.Println("Unknown command. Available commands:")
		fmt.Println("  check  - Check database integrity")
		fmt.Println("  repair - Attempt to repair database")
		fmt.Println("  backup - Create a database backup")
		os.Exit(1)
	}
}

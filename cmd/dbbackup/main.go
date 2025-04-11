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
		dbPath      = flag.String("db", "", "Path to SQLite database file (default: data/securesignin.db)")
		backupDir   = flag.String("dir", "", "Backup directory (default: [dbdir]/backups)")
		checkBefore = flag.Bool("check", true, "Run integrity check before backup")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
	)

	// Parse flags
	flag.Parse()

	// Set default database path if not provided
	if *dbPath == "" {
		*dbPath = filepath.Join("data", "securesignin.db")
	}

	// Check if database file exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("Database file not found at %s", *dbPath)
	}

	// Run integrity check if requested
	if *checkBefore {
		if *verbose {
			log.Printf("Running integrity check before backup...")
		}

		isValid, result, err := utils.CheckDatabaseIntegrity(*dbPath)
		if err != nil {
			log.Printf("Warning: Integrity check failed: %v", err)
		} else if !isValid {
			log.Printf("Warning: Database integrity check failed: %s", result)
			log.Println("Continuing with backup anyway, but the backup may be corrupted")
		} else if *verbose {
			log.Println("Integrity check passed")
		}
	}

	// Create custom backup directory if specified
	if *backupDir != "" {
		if err := os.MkdirAll(*backupDir, 0755); err != nil {
			log.Fatalf("Failed to create backup directory: %v", err)
		}

		// Copy the database file to the custom location
		dbFilename := "securesignin.db.bak" // Use the standard backup name
		backupPath := filepath.Join(*backupDir, dbFilename)

		if *verbose {
			log.Printf("Creating backup at %s...", backupPath)
		}

		// Remove existing backup if it exists
		if _, err := os.Stat(backupPath); err == nil {
			if err := os.Remove(backupPath); err != nil {
				log.Fatalf("Failed to remove existing backup file: %v", err)
			}
		}

		// Open source file
		srcFile, err := os.Open(*dbPath)
		if err != nil {
			log.Fatalf("Failed to open source database: %v", err)
		}
		defer srcFile.Close()

		// Create destination file
		destFile, err := os.Create(backupPath)
		if err != nil {
			log.Fatalf("Failed to create backup file: %v", err)
		}
		defer destFile.Close()

		// Copy file contents
		bufferSize := 1024 * 1024 // 1MB buffer
		buffer := make([]byte, bufferSize)

		for {
			bytesRead, err := srcFile.Read(buffer)
			if bytesRead > 0 {
				_, err := destFile.Write(buffer[:bytesRead])
				if err != nil {
					log.Fatalf("Failed to write to backup file: %v", err)
				}
			}

			if err != nil {
				break // End of file or error
			}
		}

		fmt.Printf("✅ Database backup created at %s\n", backupPath)
	} else {
		// Use the built-in backup utility
		backupPath, err := utils.BackupDatabase(*dbPath)
		if err != nil {
			log.Fatalf("Backup failed: %v", err)
		}

		fmt.Printf("✅ Database backup created at %s\n", backupPath)
	}

	fmt.Println("\nBackup complete. Your database is now safely backed up with a single backup file.")
}

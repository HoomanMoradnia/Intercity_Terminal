#!/usr/bin/env node

/**
 * Database Preservation Tool for Secure Sign In
 * 
 * This script backs up the database before building and restores it afterward
 * to ensure data is not lost during rebuilds.
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

// Define paths
const HOME_DIR = os.homedir();
const DB_LOCATIONS = [
  path.join(HOME_DIR, '.securesignin', 'securesignin.db'),          // Default electron location
  path.join(HOME_DIR, 'SecureSignIn', 'data', 'securesignin.db'),   // Shared location with Docker
  path.join('/app/data/securesignin.db')                            // Docker container location (may not be accessible)
];
const BACKUP_DIR = path.join(HOME_DIR, '.config', 'secure-sign-in-app', 'backups');
const TEMP_BACKUP_PATH = path.join(BACKUP_DIR, 'pre-build-backup.db');

// Process arguments
const args = process.argv.slice(2);
const operation = args[0] || 'backup'; // Default to backup

// Ensure backup directory exists
function ensureBackupDirExists() {
  if (!fs.existsSync(BACKUP_DIR)) {
    fs.mkdirSync(BACKUP_DIR, { recursive: true });
    console.log(`Created backup directory: ${BACKUP_DIR}`);
  }
}

// Ensure directory exists
function ensureDirExists(dirPath) {
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
    console.log(`Created directory: ${dirPath}`);
  }
}

// Find the first existing database file
function findExistingDatabase() {
  for (const dbPath of DB_LOCATIONS) {
    if (fs.existsSync(dbPath)) {
      console.log(`Found existing database at: ${dbPath}`);
      return dbPath;
    }
  }
  console.log("No existing database found at any known location.");
  return null;
}

// Backup the database before build
function backupDatabase() {
  const existingDb = findExistingDatabase();
  if (!existingDb) {
    console.log("No database found. Nothing to backup.");
    return false;
  }

  ensureBackupDirExists();

  try {
    fs.copyFileSync(existingDb, TEMP_BACKUP_PATH);
    console.log(`Successfully backed up database from ${existingDb} to ${TEMP_BACKUP_PATH}`);
    return true;
  } catch (error) {
    console.error(`Error backing up database: ${error.message}`);
    return false;
  }
}

// Restore the database after build
function restoreDatabase() {
  if (!fs.existsSync(TEMP_BACKUP_PATH)) {
    console.log(`No backup found at ${TEMP_BACKUP_PATH}. Nothing to restore.`);
    return false;
  }

  let restoredAny = false;

  // Restore to all possible locations
  for (const dbPath of DB_LOCATIONS) {
    try {
      // Skip Docker container path which might not be accessible
      if (dbPath.startsWith('/app/') && os.platform() !== 'linux') {
        continue;
      }
      
      // Ensure the target directory exists
      const dbDir = path.dirname(dbPath);
      ensureDirExists(dbDir);

      fs.copyFileSync(TEMP_BACKUP_PATH, dbPath);
      console.log(`Successfully restored database to: ${dbPath}`);
      restoredAny = true;
    } catch (error) {
      console.error(`Error restoring database to ${dbPath}: ${error.message}`);
    }
  }

  // Always create a permanent backup
  try {
    const permanentBackup = path.join(BACKUP_DIR, `backup-${new Date().toISOString().replace(/[:.]/g, '-')}.db`);
    fs.copyFileSync(TEMP_BACKUP_PATH, permanentBackup);
    console.log(`Created permanent backup at ${permanentBackup}`);
    
    // Also keep a current backup for easy access
    const currentBackup = path.join(BACKUP_DIR, 'current-backup.db');
    fs.copyFileSync(TEMP_BACKUP_PATH, currentBackup);
    console.log(`Updated current backup at ${currentBackup}`);
  } catch (error) {
    console.error(`Error creating permanent backup: ${error.message}`);
  }

  return restoredAny;
}

// Main function
function main() {
  if (operation === 'backup') {
    console.log('=== Backing up database before build ===');
    backupDatabase();
  } else if (operation === 'restore') {
    console.log('=== Restoring database after build ===');
    restoreDatabase();
  } else {
    console.error(`Unknown operation: ${operation}`);
    console.log('Usage: node preserve-data.js [backup|restore]');
    process.exit(1);
  }
}

// Run the main function
main(); 
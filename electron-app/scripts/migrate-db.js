#!/usr/bin/env node

/**
 * Database Migration Tool for Secure Sign In
 * 
 * This script migrates data from the web application database to the Electron app database.
 * It can be used to import data from various sources into the local SQLite database.
 */

const fs = require('fs');
const path = require('path');
const sqlite3 = require('sqlite3').verbose();
const { execSync } = require('child_process');
const os = require('os');
const readline = require('readline');

// Create readline interface for user input
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
});

// Define paths
const HOME_DIR = os.homedir();
const LOCAL_DB_PATH = path.join(HOME_DIR, '.securesignin', 'securesignin.db');
const CONFIG_DIR = path.join(HOME_DIR, '.config', 'secure-sign-in-app');
const BACKUP_DIR = path.join(CONFIG_DIR, 'backups');

function askQuestion(query) {
  return new Promise(resolve => rl.question(query, resolve));
}

async function main() {
  console.log('=== Secure Sign In Database Migration Tool ===');
  
  // Ensure directories exist
  ensureDirectoriesExist();
  
  // Check if local database exists
  if (!fs.existsSync(LOCAL_DB_PATH)) {
    console.log(`Local database not found at: ${LOCAL_DB_PATH}`);
    console.log('Creating a new database...');
    createEmptyDatabase();
  } else {
    // Backup existing database
    const backupPath = backupLocalDatabase();
    console.log(`Created backup of existing database at: ${backupPath}`);
  }
  
  // Ask user for source database type
  const dbType = await askQuestion(
    'Select source database type:\n' +
    '1. Web app SQLite database\n' +
    '2. Web app PostgreSQL database (requires pg-dump file)\n' +
    '3. SQL dump file (.sql)\n' +
    'Enter choice (1-3): '
  );
  
  let sourceDbPath = '';
  
  switch (dbType) {
    case '1':
      sourceDbPath = await askQuestion('Enter path to the web app SQLite database file: ');
      await migrateSqliteToSqlite(sourceDbPath, LOCAL_DB_PATH);
      break;
    case '2':
      console.log('PostgreSQL migration requires a SQL dump file.');
      sourceDbPath = await askQuestion('Enter path to the PostgreSQL dump file (.sql): ');
      await migrateSqlDumpToSqlite(sourceDbPath, LOCAL_DB_PATH);
      break;
    case '3':
      sourceDbPath = await askQuestion('Enter path to SQL dump file (.sql): ');
      await migrateSqlDumpToSqlite(sourceDbPath, LOCAL_DB_PATH);
      break;
    default:
      console.log('Invalid choice. Exiting.');
      process.exit(1);
  }
  
  rl.close();
}

function ensureDirectoriesExist() {
  const dirs = [
    path.dirname(LOCAL_DB_PATH),
    CONFIG_DIR,
    BACKUP_DIR
  ];
  
  dirs.forEach(dir => {
    if (!fs.existsSync(dir)) {
      console.log(`Creating directory: ${dir}`);
      fs.mkdirSync(dir, { recursive: true });
    }
  });
}

function createEmptyDatabase() {
  const db = new sqlite3.Database(LOCAL_DB_PATH);
  
  // Create basic schema
  db.serialize(() => {
    // Users table
    db.run(`
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        password TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        date_of_birth TEXT,
        social_security TEXT
      );
    `);
    
    // Login history table
    db.run(`
      CREATE TABLE IF NOT EXISTS login_history (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        login_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        ip_address TEXT,
        success INTEGER,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
      );
    `);
    
    // Create indexes
    db.run(`CREATE UNIQUE INDEX IF NOT EXISTS idx_username ON users(username);`);
  });
  
  db.close();
  console.log('Empty database created successfully.');
}

function backupLocalDatabase() {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const backupPath = path.join(BACKUP_DIR, `securesignin-backup-${timestamp}.db`);
  
  fs.copyFileSync(LOCAL_DB_PATH, backupPath);
  return backupPath;
}

async function migrateSqliteToSqlite(sourceDbPath, targetDbPath) {
  if (!fs.existsSync(sourceDbPath)) {
    console.error(`Source database does not exist: ${sourceDbPath}`);
    return;
  }
  
  try {
    console.log('Starting SQLite to SQLite migration...');
    
    const sourceDb = new sqlite3.Database(sourceDbPath, sqlite3.OPEN_READONLY);
    const targetDb = new sqlite3.Database(targetDbPath, sqlite3.OPEN_READWRITE);
    
    // Begin transaction
    targetDb.run('BEGIN TRANSACTION');
    
    // Migrate users table
    await new Promise((resolve, reject) => {
      sourceDb.all('SELECT * FROM users', (err, rows) => {
        if (err) {
          reject(err);
          return;
        }
        
        console.log(`Found ${rows.length} users to migrate.`);
        
        const stmt = targetDb.prepare(`
          INSERT OR IGNORE INTO users 
          (id, username, password, created_at, date_of_birth, social_security) 
          VALUES (?, ?, ?, ?, ?, ?)
        `);
        
        rows.forEach(row => {
          stmt.run(
            row.id, 
            row.username, 
            row.password, 
            row.created_at, 
            row.date_of_birth, 
            row.social_security
          );
        });
        
        stmt.finalize();
        resolve();
      });
    });
    
    // Migrate login history table
    await new Promise((resolve, reject) => {
      sourceDb.all('SELECT * FROM login_history', (err, rows) => {
        if (err) {
          reject(err);
          return;
        }
        
        console.log(`Found ${rows.length} login history records to migrate.`);
        
        if (rows.length > 0) {
          const stmt = targetDb.prepare(`
            INSERT OR IGNORE INTO login_history 
            (id, user_id, login_time, ip_address, success) 
            VALUES (?, ?, ?, ?, ?)
          `);
          
          rows.forEach(row => {
            stmt.run(
              row.id, 
              row.user_id, 
              row.login_time, 
              row.ip_address, 
              row.success
            );
          });
          
          stmt.finalize();
        }
        
        resolve();
      });
    });
    
    // Commit transaction
    targetDb.run('COMMIT');
    
    // Close databases
    sourceDb.close();
    targetDb.close();
    
    console.log('Migration completed successfully!');
    
  } catch (error) {
    console.error('Migration failed:', error);
  }
}

async function migrateSqlDumpToSqlite(sqlDumpPath, targetDbPath) {
  if (!fs.existsSync(sqlDumpPath)) {
    console.error(`SQL dump file does not exist: ${sqlDumpPath}`);
    return;
  }
  
  try {
    console.log('Starting SQL dump to SQLite migration...');
    
    // Convert SQL dump to SQLite compatible format
    const sqlContent = fs.readFileSync(sqlDumpPath, 'utf8');
    
    // Basic SQL parsing and conversion
    let sqliteContent = sqlContent
      // Replace PostgreSQL-specific types
      .replace(/SERIAL PRIMARY KEY/gi, 'INTEGER PRIMARY KEY AUTOINCREMENT')
      .replace(/TIMESTAMP WITH TIME ZONE/gi, 'TIMESTAMP')
      .replace(/TEXT\[\]/gi, 'TEXT')
      // Remove PostgreSQL-specific commands
      .replace(/SET .+?;/gi, '')
      .replace(/CREATE EXTENSION .+?;/gi, '')
      .replace(/ALTER TABLE .+? OWNER TO .+?;/gi, '')
      // Extract INSERT statements
      .split('\n')
      .filter(line => line.trim().startsWith('INSERT INTO') || line.trim().startsWith('CREATE TABLE'))
      .join('\n');
    
    const tempSqlPath = path.join(os.tmpdir(), 'temp_migration.sql');
    fs.writeFileSync(tempSqlPath, sqliteContent);
    
    // Execute SQL commands
    const db = new sqlite3.Database(targetDbPath);
    const sql = fs.readFileSync(tempSqlPath, 'utf8');
    
    db.serialize(() => {
      db.exec(sql, (err) => {
        if (err) {
          console.error('Error executing SQL:', err);
          console.log('Attempting to migrate data row by row...');
          
          // Fall back to row-by-row migration
          const insertStatements = sql.split('\n')
            .filter(line => line.trim().startsWith('INSERT INTO'))
            .map(line => line.trim());
          
          let successCount = 0;
          let failCount = 0;
          
          insertStatements.forEach(stmt => {
            try {
              db.run(stmt);
              successCount++;
            } catch (error) {
              failCount++;
            }
          });
          
          console.log(`Migrated ${successCount} rows successfully. Failed: ${failCount} rows.`);
        } else {
          console.log('SQL dump imported successfully.');
        }
        
        db.close();
        
        // Clean up temp file
        fs.unlinkSync(tempSqlPath);
      });
    });
    
  } catch (error) {
    console.error('Migration failed:', error);
  }
}

// Run the main function
main().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
}); 
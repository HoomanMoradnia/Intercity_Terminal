# Secure Sign In - Database Migration Guide

This guide will help you migrate data from your web application to the Electron desktop application.

## Prerequisites

- Node.js installed (version 14 or higher)
- npm installed
- If migrating from PostgreSQL: Access to a PostgreSQL dump file

## Installation

Before running the migration script, you need to install the required dependencies:

```bash
cd electron-app
npm install sqlite3
```

## Migration Options

The migration tool supports three types of migrations:

1. **SQLite to SQLite**: If your web app uses SQLite
2. **PostgreSQL to SQLite**: If your web app uses PostgreSQL
3. **SQL Dump to SQLite**: From any SQL dump file

## How to Run the Migration

1. Make sure your Electron app is not running
2. Navigate to the scripts directory:

```bash
cd electron-app/scripts
```

3. Run the migration script:

```bash
# Linux/macOS
node migrate-db.js

# Windows
node migrate-db.js
```

4. Follow the prompts in the interactive tool:
   - Choose your source database type
   - Provide the path to your source database or SQL dump file
   - Wait for the migration to complete

## Database Locations

- **Electron app database**: `~/.securesignin/securesignin.db` (Linux/macOS) or `%USERPROFILE%\.securesignin\securesignin.db` (Windows)
- **Backups**: Created automatically in `~/.config/secure-sign-in-app/backups/` (Linux/macOS) or `%USERPROFILE%\.config\secure-sign-in-app\backups\` (Windows)

## Migrating from PostgreSQL

To migrate from PostgreSQL, you first need to create a SQL dump file:

```bash
# Create a SQL dump of your PostgreSQL database
pg_dump -U your_username -d your_database_name > postgres_dump.sql
```

Then run the migration tool and select option 2 when prompted.

## Troubleshooting

If you encounter any issues during migration:

1. Check the console output for specific error messages
2. Ensure your source database is not corrupted
3. Verify the file paths you're entering
4. Make sure you have write permissions for the target locations

If the migration fails, your original data will remain unchanged, and a backup of your current database will be created before any migration attempts.

## Running the Application After Migration

After successful migration:

1. Close any running instances of the application
2. Run the application using the provided scripts:
   - Linux: `./run-app.sh` in the dist directory
   - Windows: `run-app.bat` in the dist directory

These scripts ensure proper environment setup for database access.

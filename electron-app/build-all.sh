#!/bin/bash

# Master build script for the Secure Sign In Desktop Application
# This script builds the Go backend and packages the Electron app for Linux and Windows
# Automatically installs dependencies if needed and ensures proper database setup
# Preserves user data during rebuilds

set -e  # Exit on any error

# Display current working directory
echo "Current directory: $(pwd)"

# Check if running from the electron-app directory
if [[ "$(basename "$(pwd)")" != "electron-app" ]]; then
  echo "This script must be run from the electron-app directory."
  echo "Please cd into the electron-app directory and try again."
  exit 1
fi

# Function to install Node.js and npm on Linux
install_nodejs_linux() {
  echo "Installing Node.js and npm..."
  
  # Check if we can use apt (Debian/Ubuntu)
  if command -v apt &> /dev/null; then
    sudo apt update
    sudo apt install -y nodejs npm
  # Check if we can use dnf (Fedora)
  elif command -v dnf &> /dev/null; then
    sudo dnf install -y nodejs npm
  # Check if we can use yum (CentOS/RHEL)
  elif command -v yum &> /dev/null; then
    sudo yum install -y nodejs npm
  # Check if we can use pacman (Arch)
  elif command -v pacman &> /dev/null; then
    sudo pacman -Sy nodejs npm
  else
    echo "Couldn't detect package manager. Please install Node.js manually."
    exit 1
  fi
}

# Function to install Go on Linux
install_go_linux() {
  echo "Installing Go..."
  
  # Check if we can use apt (Debian/Ubuntu)
  if command -v apt &> /dev/null; then
    sudo apt update
    sudo apt install -y golang
  # Check if we can use dnf (Fedora)
  elif command -v dnf &> /dev/null; then
    sudo dnf install -y golang
  # Check if we can use yum (CentOS/RHEL)
  elif command -v yum &> /dev/null; then
    sudo yum install -y golang
  # Check if we can use pacman (Arch)
  elif command -v pacman &> /dev/null; then
    sudo pacman -Sy go
  else
    echo "Couldn't detect package manager. Please install Go manually."
    exit 1
  fi
}

# Function to install Wine on Linux (for Windows builds)
install_wine_linux() {
  echo "Installing Wine for Windows builds..."
  
  # Check if we can use apt (Debian/Ubuntu)
  if command -v apt &> /dev/null; then
    sudo apt update
    sudo apt install -y wine64
  # Check if we can use dnf (Fedora)
  elif command -v dnf &> /dev/null; then
    sudo dnf install -y wine
  # Check if we can use yum (CentOS/RHEL)
  elif command -v yum &> /dev/null; then
    sudo yum install -y wine
  # Check if we can use pacman (Arch)
  elif command -v pacman &> /dev/null; then
    sudo pacman -Sy wine
  else
    echo "Couldn't detect package manager. Windows builds may fail without Wine."
    # Continue anyway
  fi
}

# Function to add database setup scripts
create_db_setup_scripts() {
  echo "Creating database setup scripts..."
  
  # Create Linux setup script
  cat > ../scripts/linux-db-setup.sh << 'EOF'
#!/bin/bash

# Set paths
APP_CONFIG_DIR="$HOME/.config/secure-sign-in-app"
USER_HOME_DIR="$HOME/.securesignin"
DB_PATH="$USER_HOME_DIR/securesignin.db"
KEY_PATH="$USER_HOME_DIR/encryption.key"

# Create all necessary directories
echo "Creating application directories..."
mkdir -p "$APP_CONFIG_DIR"
mkdir -p "$USER_HOME_DIR"
mkdir -p "$APP_CONFIG_DIR/backups"

# Set permissions
echo "Setting permissions..."
chmod 755 "$APP_CONFIG_DIR"
chmod 755 "$USER_HOME_DIR"
chmod 755 "$APP_CONFIG_DIR/backups"

# Check for existing encryption key
if [ ! -f "$KEY_PATH" ]; then
  echo "No encryption key found, creating placeholder (app will generate the proper key)"
  dd if=/dev/urandom bs=1 count=32 of="$KEY_PATH" 2>/dev/null
  chmod 600 "$KEY_PATH"
fi

echo "Database setup complete. Your database will be stored at: $DB_PATH"
EOF
  chmod +x ../scripts/linux-db-setup.sh
  
  # Create Windows setup script
  cat > ../scripts/windows-db-setup.bat << 'EOF'
@echo off
setlocal enabledelayedexpansion

echo === Secure Sign In Database Setup ===

:: Set paths
set "APP_CONFIG_DIR=%USERPROFILE%\.config\secure-sign-in-app"
set "USER_HOME_DIR=%USERPROFILE%\.securesignin"
set "DB_PATH=%USER_HOME_DIR%\securesignin.db"
set "KEY_PATH=%USER_HOME_DIR%\encryption.key"

:: Create all necessary directories
echo Creating application directories...
if not exist "%APP_CONFIG_DIR%" mkdir "%APP_CONFIG_DIR%"
if not exist "%USER_HOME_DIR%" mkdir "%USER_HOME_DIR%"
if not exist "%APP_CONFIG_DIR%\backups" mkdir "%APP_CONFIG_DIR%\backups"

:: Check for existing encryption key
if not exist "%KEY_PATH%" (
  echo No encryption key found, creating placeholder for app to use
  certutil -f -encodehex NUL "%KEY_PATH%" 32 >nul 2>&1
  if errorlevel 1 (
    echo Failed to create key file. Please run as administrator.
    exit /b 1
  )
)

echo Database setup complete. Your database will be stored at: %DB_PATH%
EOF
  
  # Create modified launch scripts
  cat > ../scripts/run-app.sh << 'EOF'
#!/bin/bash
# Run script for Secure Sign In application

# Set correct database path
export SQLITE_DB_PATH="$HOME/.securesignin/securesignin.db"

# Run database setup script if it exists
if [ -f "./scripts/linux-db-setup.sh" ]; then
  bash "./scripts/linux-db-setup.sh"
fi

# Run the application
APP_PATH="./Secure Sign In-1.0.0.AppImage"
if [ -f "$APP_PATH" ]; then
  echo "Starting Secure Sign In application..."
  "$APP_PATH"
else
  echo "Error: Application not found at $APP_PATH"
  echo "Please ensure you're running this script from the application directory."
  exit 1
fi
EOF
  chmod +x ../scripts/run-app.sh
  
  cat > ../scripts/run-app.bat << 'EOF'
@echo off
:: Run script for Secure Sign In application

:: Set correct database path
set "SQLITE_DB_PATH=%USERPROFILE%\.securesignin\securesignin.db"

:: Run database setup script if it exists
if exist ".\scripts\windows-db-setup.bat" (
  call ".\scripts\windows-db-setup.bat"
)

:: Run the application
set "APP_PATH=Secure Sign In.exe"
if exist "%APP_PATH%" (
  echo Starting Secure Sign In application...
  start "" "%APP_PATH%"
) else (
  echo Error: Application not found at %APP_PATH%
  echo Please ensure you're running this script from the application directory.
  exit /b 1
)
EOF

  echo "Database setup scripts created successfully."
}

# Function to backup and restore user data
preserve_user_data() {
  echo "Creating data preservation tool..."
  
  mkdir -p scripts
  
  # Create the preservation script if it doesn't exist yet
  if [ ! -f "scripts/preserve-data.js" ]; then
    echo "Creating data preservation script..."
    cat > scripts/preserve-data.js << 'EOF'
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
const DB_PATH = path.join(HOME_DIR, '.securesignin', 'securesignin.db');
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

// Backup the database before build
function backupDatabase() {
  if (!fs.existsSync(DB_PATH)) {
    console.log(`No database found at ${DB_PATH}. Nothing to backup.`);
    return false;
  }

  ensureBackupDirExists();

  try {
    fs.copyFileSync(DB_PATH, TEMP_BACKUP_PATH);
    console.log(`Successfully backed up database to ${TEMP_BACKUP_PATH}`);
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

  // Ensure the target directory exists
  const dbDir = path.dirname(DB_PATH);
  if (!fs.existsSync(dbDir)) {
    fs.mkdirSync(dbDir, { recursive: true });
    console.log(`Created database directory: ${dbDir}`);
  }

  try {
    fs.copyFileSync(TEMP_BACKUP_PATH, DB_PATH);
    console.log(\`Successfully restored database from \${TEMP_BACKUP_PATH} to \${DB_PATH}\`);
    
    // Create a single permanent backup file
    const permanentBackup = path.join(BACKUP_DIR, \`current-backup.db\`);
    fs.copyFileSync(TEMP_BACKUP_PATH, permanentBackup);
    console.log(\`Updated permanent backup at \${permanentBackup}\`);
    
    return true;
  } catch (error) {
    console.error(`Error restoring database: ${error.message}`);
    return false;
  }
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
EOF
    chmod +x scripts/preserve-data.js
  fi
  
  echo "Data preservation tool created successfully."
}

# Check if Node.js and npm are installed, install if not
if ! command -v node &> /dev/null || ! command -v npm &> /dev/null; then
  echo "Node.js or npm not found. Installing..."
  install_nodejs_linux
fi

# Check if Go is installed, install if not
if ! command -v go &> /dev/null; then
  echo "Go not found. Installing..."
  install_go_linux
fi

# Display versions
echo "Node.js version: $(node --version)"
echo "npm version: $(npm --version)"
echo "Go version: $(go version)"

# Make sure scripts directory exists
mkdir -p ../scripts

# Create database setup scripts
create_db_setup_scripts

# Create and run the data preservation tool
preserve_user_data

# Backup existing database before build
echo "Backing up user data before build..."
node scripts/preserve-data.js backup

# Modify package.json to include database environment variable setup
if grep -q "scripts/run-app" package.json; then
  echo "Package.json already updated with startup scripts."
else
  echo "Updating package.json to include database environment setup..."
  # Create a temporary file with the updated content
  sed -i 's/"extraResources": \[/"extraResources": [\n      {\n        "from": "..\/scripts",\n        "to": "scripts",\n        "filter": ["**\/*"]\n      },/g' package.json
fi

# Install dependencies
echo "Installing Node.js dependencies..."
npm install
if [ $? -ne 0 ]; then
  echo "Failed to install Node.js dependencies."
  exit 1
fi
echo "Dependencies installed successfully."

# Build the Go backend
echo "Building Go backend for Linux and Windows..."
chmod +x build-backend.sh
./build-backend.sh
if [ $? -ne 0 ]; then
  echo "Failed to build Go backend."
  exit 1
fi
echo "Go backend built successfully."

# Check which platforms to build for
BUILD_LINUX=1
BUILD_WINDOWS=1

# Check if Wine is installed if building for Windows, install if not found
if [ $BUILD_WINDOWS -eq 1 ] && ! command -v wine &> /dev/null; then
  echo "Wine not found. Installing Wine for Windows builds..."
  install_wine_linux
fi

# Package the application
echo "Packaging the application..."

if [ $BUILD_LINUX -eq 1 ] && [ $BUILD_WINDOWS -eq 1 ]; then
  # Build for both platforms
  echo "Building for both Linux and Windows..."
  npm run package-all
elif [ $BUILD_LINUX -eq 1 ]; then
  # Build for Linux only
  echo "Building for Linux only..."
  npm run package-linux
elif [ $BUILD_WINDOWS -eq 1 ]; then
  # Build for Windows only
  echo "Building for Windows only..."
  npm run package-win
else
  echo "No platforms selected for building. Exiting."
  exit 1
fi

if [ $? -ne 0 ]; then
  echo "Failed to package the application."
  exit 1
fi

echo "Application packaged successfully."
echo "The packaged applications can be found in the dist/ directory."

# Copy run scripts to dist directory
echo "Copying run scripts to dist directory..."
cp ../scripts/run-app.sh dist/
cp ../scripts/run-app.bat dist/
cp ../scripts/linux-db-setup.sh dist/
cp ../scripts/windows-db-setup.bat dist/
cp scripts/preserve-data.js dist/scripts/

# Add instructions
echo "Creating post-build README..."
cat > dist/README.txt << 'EOF'
Secure Sign In Application
=========================

To ensure proper database setup and permissions:

For Linux users:
1. Run the application using the provided script:
   ./run-app.sh

For Windows users:
1. Run the application using the provided script:
   run-app.bat

If you experience database errors:
- Linux: Run linux-db-setup.sh script to reset permissions
- Windows: Run windows-db-setup.bat as administrator to reset permissions

Your data will be stored in:
- Linux: ~/.securesignin/securesignin.db
- Windows: %USERPROFILE%\.securesignin\securesignin.db

Data Preservation:
- If you need to reinstall, your data is automatically backed up
- Backups are stored in ~/.config/secure-sign-in-app/backups (Linux) or
  %USERPROFILE%\.config\secure-sign-in-app\backups (Windows)
EOF

# Restore the database after build
echo "Restoring user data after build..."
node scripts/preserve-data.js restore

# List the contents of the dist directory
echo "Contents of the dist directory:"
ls -la dist/

echo "Build process completed successfully!"
echo "Please use the run-app.sh (Linux) or run-app.bat (Windows) scripts to ensure proper database setup."
echo "Your data has been preserved after the build." 
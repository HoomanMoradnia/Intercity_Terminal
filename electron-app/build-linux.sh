#!/bin/bash

# Build script for Secure Sign In application on Linux
# Including data preservation to prevent data loss during rebuilds

echo "Building Secure Sign In for Linux..."
echo "Current directory: $(pwd)"

# Ensure we're in the electron-app directory
if [[ "$(basename "$(pwd)")" != "electron-app" ]]; then
  echo "This script must be run from the electron-app directory."
  echo "Please cd into the electron-app directory and try again."
  exit 1
fi

# Create data preservation script
echo "Creating data preservation script..."
mkdir -p scripts

# Create the preserve-data.js script
if [ ! -f "scripts/preserve-data.js" ]; then
  cat > scripts/preserve-data.js << 'EOF'
// Database Preservation Tool
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
    console.log(`Successfully restored database to ${DB_PATH}`);

    // Make a timestamp backup as well
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const timestampBackup = path.join(BACKUP_DIR, `backup-${timestamp}.db`);
    fs.copyFileSync(TEMP_BACKUP_PATH, timestampBackup);
    console.log(`Created timestamped backup at ${timestampBackup}`);

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
fi

# Backup user data before building
echo "Backing up user data before build..."
node scripts/preserve-data.js backup

# Create scripts directory in project root
mkdir -p ../scripts

# Create Linux setup script in project root scripts
if [ ! -f "../scripts/linux-db-setup.sh" ]; then
  cat > ../scripts/linux-db-setup.sh << 'EOF'
#!/bin/bash

echo "=== Secure Sign In Database Setup ==="

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

# Set directory permissions
chmod 755 "$APP_CONFIG_DIR"
chmod 755 "$USER_HOME_DIR"
chmod 755 "$APP_CONFIG_DIR/backups"

# Check for existing encryption key
if [ ! -f "$KEY_PATH" ]; then
  echo "No encryption key found, creating placeholder for app to use"
  dd if=/dev/urandom bs=32 count=1 of="$KEY_PATH" 2>/dev/null
  if [ $? -ne 0 ]; then
    echo "Failed to create key file. Please check permissions."
    exit 1
  fi
  chmod 600 "$KEY_PATH"
fi

echo "Database setup complete. Your database will be stored at: $DB_PATH"
EOF
  chmod +x ../scripts/linux-db-setup.sh
fi

# Create run script in project root scripts
if [ ! -f "../scripts/run-app.sh" ]; then
  cat > ../scripts/run-app.sh << 'EOF'
#!/bin/bash

# Run script for Secure Sign In application

# Set correct database path
export SQLITE_DB_PATH="$HOME/.securesignin/securesignin.db"

# Run database setup script if it exists
if [ -f "./scripts/linux-db-setup.sh" ]; then
  ./scripts/linux-db-setup.sh
fi

# Run the application
if [ -f "./Secure Sign In-1.0.0.AppImage" ]; then
  echo "Starting Secure Sign In application..."
  ./Secure\ Sign\ In-1.0.0.AppImage
else
  echo "Error: Application not found."
  echo "Please ensure you're running this script from the application directory."
  exit 1
fi
EOF
  chmod +x ../scripts/run-app.sh
fi

# Build the backend
echo "Building Go backend..."
cd ..
GOOS=linux GOARCH=amd64 go build -o main .
if [ $? -ne 0 ]; then
  echo "Failed to build Go backend."
  exit 1
fi
echo "Go backend built successfully."

# Return to electron-app directory
cd electron-app

# Install dependencies
echo "Installing Node.js dependencies..."
npm install
if [ $? -ne 0 ]; then
  echo "Failed to install dependencies."
  exit 1
fi
echo "Dependencies installed successfully."

# Package the application
echo "Packaging the application..."
npm run package-linux
if [ $? -ne 0 ]; then
  echo "Failed to package the application."
  exit 1
fi
echo "Application packaged successfully."

# Copy scripts to the dist directory
echo "Copying scripts to dist directory..."
mkdir -p dist/scripts
cp ../scripts/linux-db-setup.sh dist/
cp ../scripts/run-app.sh dist/
cp scripts/preserve-data.js dist/scripts/
chmod +x dist/linux-db-setup.sh
chmod +x dist/run-app.sh

# Create a README file
echo "Creating README..."
cat > dist/README.txt << EOF
Secure Sign In Application
=========================

To ensure proper database setup and permissions:

For Linux users:
1. Run the application using the provided script:
   ./run-app.sh

If you experience database errors:
- Run linux-db-setup.sh to reset permissions

Your data will be stored in:
- $HOME/.securesignin/securesignin.db

Data Preservation:
- If you need to reinstall, your data is automatically backed up
- Backups are stored in $HOME/.config/secure-sign-in-app/backups
EOF

# Restore user data after building
echo "Restoring user data after build..."
node scripts/preserve-data.js restore

echo "Build process completed successfully!"
echo "Please use run-app.sh to start the application with proper database configuration."
echo "Your data has been preserved through the build process." 
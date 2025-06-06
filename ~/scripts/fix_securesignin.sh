#!/bin/bash

# Fix script for Secure Sign In app
echo "=== Secure Sign In Database Fix ==="

# Set paths
APP_CONFIG_DIR="$HOME/.config/secure-sign-in-app"
DB_PATH="$APP_CONFIG_DIR/securesignin.db"
USER_HOME_DIR="$HOME/.securesignin"

# Create all necessary directories with proper permissions
echo "Creating application directories..."
mkdir -p "$APP_CONFIG_DIR"
mkdir -p "$USER_HOME_DIR"

# Set permissions
echo "Setting permissions..."
chmod 777 "$APP_CONFIG_DIR"
chmod 777 "$USER_HOME_DIR"

# Check if database exists
if [ -f "$DB_PATH" ]; then
  echo "Database exists at: $DB_PATH"
  echo "Testing database..."
  
  # Try to remove it if it's corrupted
  if sqlite3 "$DB_PATH" "PRAGMA integrity_check;" &>/dev/null; then
    echo "Database appears valid."
  else
    echo "Database is corrupted, removing..."
    rm -f "$DB_PATH"
    echo "Database removed."
  fi
else
  echo "No database found at: $DB_PATH"
fi

# Create key directory
echo "Creating key directory..."
mkdir -p "$USER_HOME_DIR"

# Check for existing encryption key
KEY_PATH="$USER_HOME_DIR/encryption.key"
if [ ! -f "$KEY_PATH" ]; then
  echo "No encryption key found, creating a placeholder (app will generate the actual key)"
  dd if=/dev/urandom bs=1 count=32 of="$KEY_PATH" 2>/dev/null
  chmod 600 "$KEY_PATH"
fi

echo "=== Fix complete ==="
echo "Now try running the application again."
echo "If issues persist, you may need to launch the application with:"
echo "SQLITE_DB_PATH=\"$HOME/.securesignin/securesignin.db\" ./\"Secure Sign In-1.0.0.AppImage\"" 
#!/bin/bash

# Secure Sign In - Shared Database Setup
# This script sets up the shared database environment for both Docker and Electron apps

echo "=== Setting up shared database environment for Secure Sign In ==="

# Define paths
HOME_DIR="$HOME"
SHARED_DATA_DIR="$HOME_DIR/.SecureSignIn/data"
ELECTRON_DATA_DIR="$HOME_DIR/.securesignin"
ELECTRON_DB_PATH="$ELECTRON_DATA_DIR/securesignin.db"
SHARED_DB_PATH="$SHARED_DATA_DIR/securesignin.db"
APP_CONFIG_DIR="$HOME_DIR/.config/secure-sign-in-app"
BACKUP_DIR="$APP_CONFIG_DIR/backups"
KEY_PATH="$ELECTRON_DATA_DIR/encryption.key"

# Create all necessary directories
echo "Creating application directories..."
mkdir -p "$APP_CONFIG_DIR"
mkdir -p "$ELECTRON_DATA_DIR"
mkdir -p "$BACKUP_DIR"
mkdir -p "$SHARED_DATA_DIR"

# Set appropriate permissions
echo "Setting proper permissions..."
chmod 755 "$APP_CONFIG_DIR"
chmod 755 "$ELECTRON_DATA_DIR"
chmod 755 "$BACKUP_DIR"
chmod 755 "$SHARED_DATA_DIR"

# If there's an existing database in the Electron directory but not in the shared directory, copy it
if [ -f "$ELECTRON_DB_PATH" ] && [ ! -f "$SHARED_DB_PATH" ]; then
  echo "Found existing database in Electron directory, copying to shared location..."
  cp "$ELECTRON_DB_PATH" "$SHARED_DB_PATH"
  echo "Creating backup of original database..."
  cp "$ELECTRON_DB_PATH" "$BACKUP_DIR/original-electron-db.bak"
  echo "The original database is preserved at: $BACKUP_DIR/original-electron-db.bak"
fi

# Ensure encryption key exists
if [ ! -f "$KEY_PATH" ]; then
  echo "No encryption key found, creating a new one..."
  if [ "$(uname)" == "Darwin" ] || [ "$(uname)" == "Linux" ]; then
    # macOS or Linux
    dd if=/dev/urandom bs=32 count=1 of="$KEY_PATH" 2>/dev/null
  else
    # Windows with WSL or Git Bash
    echo "Please run the Windows setup script instead."
    exit 1
  fi
  
  if [ $? -ne 0 ]; then
    echo "Failed to create key file. Please check permissions."
    exit 1
  fi
  chmod 600 "$KEY_PATH"
fi

echo "Creating a symbolic link from Electron directory to shared database..."
if [ -f "$ELECTRON_DB_PATH" ]; then
  mv "$ELECTRON_DB_PATH" "$ELECTRON_DB_PATH.bak"
  echo "Backed up existing database to $ELECTRON_DB_PATH.bak"
fi

# Create a symbolic link so Electron app can find the database in its original location
ln -sf "$SHARED_DB_PATH" "$ELECTRON_DB_PATH"

echo ""
echo "====== Setup Complete ======"
echo "Shared database location: $SHARED_DB_PATH"
echo "This database will be used by both Docker and Electron applications."
echo ""
echo "To run the Docker container with this setup:"
echo "  ./run.sh restart"
echo ""
echo "To run the Electron app with this setup:"
echo "  cd electron-app"
echo "  npm start"
echo "==============================" 
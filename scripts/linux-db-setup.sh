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

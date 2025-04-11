#!/bin/bash

# Run standalone script for SecureSignIn - suitable for AppImage packaging
# This script sets up the environment for the application to run without Docker

# Set up environment variables
APP_DIR="$(dirname "$(readlink -f "$0")")"
export SQLITE_DB_PATH="$HOME/.securesignin/securesignin.db"
DATA_DIR="$(dirname "$SQLITE_DB_PATH")"

# Create data directory if it doesn't exist
mkdir -p "$DATA_DIR"

# Check if database file exists
if [ ! -f "$SQLITE_DB_PATH" ]; then
  echo "Creating new database at $SQLITE_DB_PATH"
fi

# Run the application
echo "Starting SecureSignIn from $APP_DIR"
echo "Using database: $SQLITE_DB_PATH"

# Kill existing instances if any
pkill -f securesignin || true

# Start the application
exec "$APP_DIR/securesignin" 
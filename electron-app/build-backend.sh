#!/bin/bash

# Script to build the Go backend for both Linux and Windows

set -e

# Get the absolute path of the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "Project root directory: $PROJECT_ROOT"

# Navigate to project root
cd "$PROJECT_ROOT"

# Build for the current platform (Linux)
echo "Building Go backend for Linux..."
go build -o main .

# Build for Windows
echo "Building Go backend for Windows..."
GOOS=windows GOARCH=amd64 go build -o main.exe .

echo "Backend builds completed successfully!" 
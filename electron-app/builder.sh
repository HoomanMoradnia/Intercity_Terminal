#!/bin/bash

# Function to detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        CYGWIN*|MINGW*|MSYS*) echo "windows";;
        *)          echo "unknown";;
    esac
}

# Function to build for Linux
build_linux() {
    echo "Building for Linux..."
    ./build-linux.sh
}

# Function to build for Windows
build_windows() {
    echo "Building for Windows..."
    ./build-windows.sh
}

# Main script
echo "Starting build process..."

# Detect OS
OS_TYPE=$(detect_os)
echo "Detected OS: $OS_TYPE"

# Build based on OS
case "$OS_TYPE" in
    "linux")
        build_linux
        ;;
    "windows")
        build_windows
        ;;
    *)
        echo "Unsupported operating system: $OS_TYPE"
        echo "Please run the appropriate build script manually:"
        echo "- For Linux: ./build-linux.sh"
        echo "- For Windows: ./build-windows.sh"
        exit 1
        ;;
esac

echo "Build process completed!" 
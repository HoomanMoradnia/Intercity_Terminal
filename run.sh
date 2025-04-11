#!/bin/bash

# Script to run the SecureSignIn application using Docker Compose
# 
# The application uses the Docker Hub image: arvinzaheri/securesignin:latest
# You can pull this image directly without building locally:
#   docker pull arvinzaheri/securesignin:latest
#
# WINDOWS USERS: This script must be run in WSL (Windows Subsystem for Linux).
# To run this script on Windows:
# 1. Install WSL (https://docs.microsoft.com/en-us/windows/wsl/install)
# 2. Open a WSL terminal
# 3. Navigate to your project directory: cd /mnt/c/path/to/project
# 4. Run this script: ./run.sh

# Function to install Docker if not present
install_docker() {
    echo "Docker is not installed. Attempting to install Docker..."
    
    # Detect the OS
    if [ -f /etc/os-release ]; then
        # freedesktop.org and systemd
        . /etc/os-release
        OS=$NAME
    elif type lsb_release >/dev/null 2>&1; then
        # linuxbase.org
        OS=$(lsb_release -si)
    elif [ -f /etc/lsb-release ]; then
        # For some versions of Debian/Ubuntu without lsb_release command
        . /etc/lsb-release
        OS=$DISTRIB_ID
    else
        # Fall back to uname, e.g. "Linux <version>", also works for BSD, etc.
        OS=$(uname -s)
    fi
    
    # Install Docker based on the detected OS
    case "$OS" in
        *Ubuntu*|*Debian*)
            echo "Detected Ubuntu/Debian. Installing Docker using apt..."
            # Update package index
            sudo apt-get update
            # Install prerequisites
            sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common
            # Add Docker's official GPG key
            curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
            # Add Docker apt repository
            sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
            # Update apt package index
            sudo apt-get update
            # Install Docker CE
            sudo apt-get install -y docker-ce docker-ce-cli containerd.io
            # Install Docker Compose
            sudo apt-get install -y docker-compose
            ;;
        *Fedora*|*CentOS*|*Red\ Hat*)
            echo "Detected Fedora/CentOS/RHEL. Installing Docker using dnf/yum..."
            # Install required packages
            sudo dnf -y install dnf-plugins-core
            # Add the Docker repository
            sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
            # Install Docker packages
            sudo dnf -y install docker-ce docker-ce-cli containerd.io
            # Install Docker Compose
            sudo dnf -y install docker-compose
            # Start and enable Docker service
            sudo systemctl start docker
            sudo systemctl enable docker
            ;;
        *Arch*)
            echo "Detected Arch Linux. Installing Docker using pacman..."
            # Install Docker and Docker Compose
            sudo pacman -Sy docker docker-compose
            # Start and enable Docker service
            sudo systemctl start docker
            sudo systemctl enable docker
            ;;
        *)
            echo "Unsupported OS for automatic Docker installation: $OS"
            echo "Please install Docker manually following the official documentation:"
            echo "https://docs.docker.com/engine/install/"
            exit 1
            ;;
    esac
    
    # Add the current user to the docker group to avoid using sudo
    echo "Adding current user to the docker group..."
    sudo usermod -aG docker $USER
    
    echo "Docker installation completed."
    echo "NOTE: You may need to log out and log back in for group changes to take effect."
    echo "Alternatively, you can continue by running the following command:"
    echo "  newgrp docker"
    
    # Check if Docker was successfully installed
    if ! command -v docker &> /dev/null; then
        echo "Docker installation failed. Please install Docker manually."
        exit 1
    fi
    
    echo "Docker installed successfully."
    
    # Try to start the Docker service if it's not running
    if ! systemctl is-active --quiet docker; then
        echo "Starting Docker service..."
        sudo systemctl start docker
    fi
}

# Function to check if Docker is installed
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo "Docker is not installed."
        read -p "Would you like to install Docker now? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            install_docker
        else
            echo "Docker installation cancelled. Please install Docker manually."
            exit 1
        fi
    fi
}

# Function to install Docker Compose if not present
install_docker_compose() {
    echo "Docker Compose is not installed. Attempting to install Docker Compose..."
    
    # Get the latest Docker Compose version
    COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
    
    # Download and install Docker Compose
    sudo curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    
    # Create a symbolic link to /usr/bin
    sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    
    # Check if Docker Compose was successfully installed
    if ! command -v docker-compose &> /dev/null; then
        echo "Docker Compose installation failed. Please install Docker Compose manually."
        exit 1
    fi
    
    echo "Docker Compose installed successfully."
}

# Function to check if Docker Compose is installed
check_docker_compose() {
    if ! command -v docker compose &> /dev/null; then
        if ! command -v docker-compose &> /dev/null; then
            echo "Docker Compose is not installed."
            read -p "Would you like to install Docker Compose now? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                install_docker_compose
            else
                echo "Docker Compose installation cancelled. Please install Docker Compose manually."
                exit 1
            fi
        fi
    fi
}

# Function to check if the application is already running
check_running() {
    if docker compose ps | grep -q "securesignin-app-1"; then
        echo "Application is already running. Use 'docker compose down' to stop it first."
        exit 1
    fi
}

# Function to display help
show_help() {
    echo "Usage: ./run.sh [command]"
    echo ""
    echo "Commands:"
    echo "  start     Start the application"
    echo "  stop      Stop the application"
    echo "  restart   Restart the application"
    echo "  logs      View application logs"
    echo "  setup     Set up shared database environment"
    echo "  help      Show this help message"
    echo ""
    echo "Default command is 'start'"
}

# Main script
case "$1" in
    "stop")
        check_docker_compose
        echo "Stopping application..."
        docker compose down
        echo "Application stopped successfully."
        ;;
    "restart")
        check_docker_compose
        echo "Restarting application..."
        docker compose down
        
        # Set up shared database environment
        if [ -f "./setup-shared-db.sh" ]; then
            echo "Setting up shared database environment..."
            ./setup-shared-db.sh
        fi
        
        docker compose up -d --build
        echo "Application restarted successfully."
        ;;
    "logs")
        check_docker_compose
        echo "Showing application logs..."
        docker compose logs -f app
        ;;
    "setup")
        if [ -f "./setup-shared-db.sh" ]; then
            echo "Setting up shared database environment..."
            ./setup-shared-db.sh
        else
            echo "Error: setup-shared-db.sh script not found."
            exit 1
        fi
        ;;
    "help"|"--help"|"-h")
        show_help
        ;;
    "start"|"")
        check_docker
        check_docker_compose
        check_running
        
        # Set up shared database environment
        if [ -f "./setup-shared-db.sh" ]; then
            echo "Setting up shared database environment..."
            ./setup-shared-db.sh
        fi
        
        echo "Starting application..."
        docker compose up -d --build
        echo "Waiting for application to become healthy..."
        # Wait up to 60 seconds for the app service to be healthy
        for i in {1..12}; do
            if docker compose ps app | grep -q '\(healthy\)'; then
                echo "Application is healthy."
                break
            fi
            echo "Still waiting for app service... ($i/12)"
            sleep 5
        done

        if ! docker compose ps app | grep -q '\(healthy\)'; then
            echo "Application did not become healthy after 60 seconds."
            docker compose logs app
            exit 1
        fi

        echo "Testing /health endpoint from within the container..."
        # Use docker compose exec to run the check inside the app container
        if docker compose exec app curl -f http://localhost:8080/health; then
            echo "Endpoint /health test successful (from container)."
        else
            echo "Endpoint /health test failed (from container)."
            docker compose logs app
            exit 1
        fi

        echo "Application started successfully."
        echo "Access the application at http://localhost:8080"
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac 
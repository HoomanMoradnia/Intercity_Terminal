# Prerequisites for Building the Desktop Application

Before you can build the Secure Sign In Desktop Application, you need to install several tools. This guide will help you set up your development environment.

## Required Tools

1. **Node.js and npm** - Required for Electron development
2. **Go** - Required for building the backend
3. **Wine** (optional, Linux only) - Required for building Windows packages on Linux

## Installing Node.js and npm

### On Linux

#### Using apt (Debian, Ubuntu)

```bash
# Update package index
sudo apt update

# Install Node.js and npm
sudo apt install nodejs npm

# Verify installation
node --version
npm --version
```

#### Using dnf (Fedora)

```bash
# Install Node.js and npm
sudo dnf install nodejs npm

# Verify installation
node --version
npm --version
```

#### Using NVM (Node Version Manager) - Recommended

NVM allows you to install and manage multiple Node.js versions.

```bash
# Install NVM
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.3/install.sh | bash

# Restart your terminal or source your profile
source ~/.bashrc  # or source ~/.zshrc if using zsh

# Install the latest LTS version of Node.js
nvm install --lts

# Verify installation
node --version
npm --version
```

### On Windows

1. Download the installer from the [official Node.js website](https://nodejs.org/)
2. Run the installer and follow the installation wizard
3. Verify installation by opening Command Prompt and running:
   ```
   node --version
   npm --version
   ```

## Installing Go

### On Linux

#### Using apt (Debian, Ubuntu)

```bash
# Update package index
sudo apt update

# Install Go
sudo apt install golang-go

# Verify installation
go version
```

#### Using dnf (Fedora)

```bash
# Install Go
sudo dnf install golang

# Verify installation
go version
```

#### Manual installation (latest version)

```bash
# Download the latest Go (adjust version as needed)
wget https://go.dev/dl/go1.22.2.linux-amd64.tar.gz

# Extract it to /usr/local
sudo tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz

# Add Go to your PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc  # or source ~/.zshrc if using zsh

# Verify installation
go version
```

### On Windows

1. Download the installer from the [official Go website](https://go.dev/dl/)
2. Run the installer and follow the installation wizard
3. Verify installation by opening Command Prompt and running:
   ```
   go version
   ```

## Installing Wine (Linux only, for Windows builds)

Wine is only needed if you want to build Windows packages from a Linux system.

### Using apt (Debian, Ubuntu)

```bash
# Add the WineHQ repository (Ubuntu)
sudo apt update
sudo apt install -y software-properties-common
sudo apt-add-repository 'deb https://dl.winehq.org/wine-builds/ubuntu/ $(lsb_release -cs) main'
wget -qO- https://dl.winehq.org/wine-builds/winehq.key | sudo apt-key add -

# Install Wine
sudo apt update
sudo apt install --install-recommends winehq-stable

# Verify installation
wine --version
```

### Using dnf (Fedora)

```bash
# Enable the Wine repository
sudo dnf config-manager --add-repo https://dl.winehq.org/wine-builds/fedora/$(rpm -E %fedora)/winehq.repo

# Install Wine
sudo dnf install winehq-stable

# Verify installation
wine --version
```

## Verifying Your Setup

After installing all the required tools, verify that everything is properly set up:

```bash
# Check Node.js and npm
node --version
npm --version

# Check Go
go version

# Check Wine (Linux only, for Windows builds)
wine --version
```

If all these commands return version information, your development environment is properly set up, and you can proceed to build the desktop application.

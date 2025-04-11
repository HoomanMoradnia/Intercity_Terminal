# Shared Database Configuration for Secure Sign In

This guide explains how to use the shared database configuration which allows both the Docker container and Electron desktop application to access the same SQLite database file.

## Overview

By default, the Docker container and Electron app use separate database locations:

- **Docker**: Stores data in a Docker volume at `/app/data/securesignin.db`
- **Electron**: Stores data in `~/.securesignin/securesignin.db` (user's home directory)

This configuration creates a shared database location that both applications can access:

- **Shared Location**: `~/.SecureSignIn/data/securesignin.db`

## Setup Instructions

### On Linux/macOS:

1. Run the setup script:

   ```bash
   ./setup-shared-db.sh
   ```

2. Start the Docker container:

   ```bash
   ./run.sh start
   ```

3. In a separate terminal, run the Electron app:
   ```bash
   cd electron-app
   npm start
   ```

### On Windows:

1. Run the setup script as administrator (right-click, "Run as administrator"):

   ```
   setup-shared-db.bat
   ```

2. Start the Docker container:

   ```
   docker-compose up -d
   ```

3. In a separate terminal, run the Electron app:
   ```
   cd electron-app
   npm start
   ```

## How It Works

The setup script performs these actions:

1. Creates a shared directory structure at `~/.SecureSignIn/data/`
2. Migrates any existing database to the shared location
3. Creates symbolic links (or copies on Windows) to maintain compatibility
4. Configures both apps to access the shared database

The Docker container mounts the shared directory from your host system, while the Electron app is configured to look for the database in the shared location first.

## Important Notes

- **Windows Symbolic Links**: Creating symbolic links on Windows requires administrator privileges. If you run the script without administrator rights, it will create a copy instead of a symbolic link, which won't stay synchronized between the two applications.

- **Existing Data**: If you have existing data in both applications, the setup script will prioritize data from the Electron app and create backups. Check the backup locations if you need to restore data.

- **Docker Volume**: The Docker Compose file has been updated to mount the shared directory instead of using a Docker volume. If you have important data in the previous Docker volume, you should back it up first.

## Troubleshooting

### Database Not Syncing

1. Make sure both apps are properly configured to use the shared location
2. Check file permissions on the shared directory
3. On Windows, verify that symbolic links were created successfully

### Permission Denied Errors

1. Check the ownership and permissions of the shared directory
2. For Docker, make sure the mounted volume has proper permissions

### Corrupted Database

1. Stop both applications
2. Check the backup directory at `~/.config/secure-sign-in-app/backups/`
3. Restore a backup to the shared location

## Building Applications

When building the applications for distribution, the setup scripts ensure the shared database configuration is maintained:

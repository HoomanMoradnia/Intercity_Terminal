# Secure Sign In Desktop Application

This is a desktop version of the Secure Sign In web application, packaged using Electron.

## Prerequisites

Before building or running the application, ensure you have the following installed:

- Node.js (v14 or higher)
- npm (v6 or higher)
- Go (v1.16 or higher) - required for building the backend

For detailed installation instructions for these prerequisites, see [PREREQUISITES.md](PREREQUISITES.md).

## Recent Changes

- **UI Flow Improvements**: The application now loads the root URL (/) instead of directly loading the login page (/login). This ensures a consistent user experience between the web version and the desktop application.
- **Shared Database Support**: The application now supports using the same database as the Docker container version, allowing for seamless data synchronization.

## Development

### Setup

1. Install dependencies:

```bash
npm install
```

2. Build the Go backend for both Linux and Windows:

```bash
# First make the build script executable
chmod +x build-backend.sh

# Then run it
./build-backend.sh
```

3. Start the application in development mode:

```bash
npm start
```

### Building

To build the application for different platforms:

#### For Linux:

```bash
npm run package-linux
```

This will create both AppImage and .deb packages in the `dist` directory.

#### For Windows:

```bash
npm run package-win
```

This will create both a portable executable and an NSIS installer in the `dist` directory.

#### For both platforms:

```bash
npm run package-all
```

## Data Preservation

The application includes a robust system to ensure your data is not lost during rebuilds or updates.

### How It Works

- When you run the build scripts (`build-linux.sh`, `build-windows.bat`), your database is automatically backed up
- After the build completes, your data is restored to the new version
- The system is intelligent enough to find your database regardless of which version you were using before (Docker, standalone, or Electron)
- Timestamped backups are created in case you need to restore a previous state

### Backup Locations

- Primary database: `$HOME/.securesignin/securesignin.db` (Linux/macOS) or `%USERPROFILE%\.securesignin\securesignin.db` (Windows)
- Backup directory: `$HOME/.config/secure-sign-in-app/backups/` (Linux/macOS) or `%USERPROFILE%\.config\secure-sign-in-app\backups\` (Windows)

### Shared Database with Docker

This application now supports using the same database as the Docker container version, allowing for seamless data synchronization between the desktop app and server versions.

#### Shared Database Location

- Linux/macOS: `~/.SecureSignIn/data/securesignin.db`
- Windows: `%USERPROFILE%\.SecureSignIn\data\securesignin.db`

#### Setting Up Shared Database

1. Run the setup script from the project root:

   ```bash
   # Linux/macOS
   ./setup-shared-db.sh

   # Windows (run as administrator)
   setup-shared-db.bat
   ```

2. This creates a shared database location and configures both the Electron app and Docker container to use it.

For more detailed information, see [SHARED_DATABASE.md](../SHARED_DATABASE.md) in the project root.

### Manual Restore

If you ever need to manually restore a backup:

```bash
# Linux/macOS
cp ~/.config/secure-sign-in-app/backups/backup-TIMESTAMP.db ~/.securesignin/securesignin.db

# Windows
copy %USERPROFILE%\.config\secure-sign-in-app\backups\backup-TIMESTAMP.db %USERPROFILE%\.securesignin\securesignin.db
```

## Application Structure

- `src/main.js`: Main Electron process that starts the application
- `backend/`: Contains the Go backend application (included from parent directory)
- `scripts/`: Contains utility scripts including the data preservation tool

## How It Works

The application embeds the Go backend executable and runs it as a child process when the Electron app starts. The frontend is then loaded in an Electron BrowserWindow, connecting to the backend running on localhost:8080.

## Troubleshooting

- **Backend fails to start**: Check the application logs. You can enable extra logging by setting the environment variable `DEBUG=1`.
- **Window shows blank screen**: The backend may not have started successfully. Check the application logs.
- **Data missing after update**: Your data might be in a different location. Check the backup directory and try a manual restore.

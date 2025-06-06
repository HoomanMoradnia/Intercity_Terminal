# Intercity Portal

A management application for an intercity terminal, built with Go and featuring a desktop version using Electron. This project originated as a secure authentication system (SecureSignIn) and is currently under development.

**Current Status:** The application currently only features the basic login/authentication pages. Core management functionality is yet to be implemented.

## Features (Planned & Existing)

- Secure user authentication (Existing)
- Password recovery system (Existing)
- Modern web interface (Partially implemented)
- Desktop application support via Electron (Setup complete)
- Docker support for easy deployment (Existing)
- Database integration (Existing)
- Automatic data preservation during updates (New)
- Terminal Management Features (Planned)

## Recent Updates

- **Authentication Improvements**: Implemented proper session management with cookies to fix login persistence and prevent "You must be logged in" errors.
- **Security Question Management**: Fixed issues with security question updates by using cookie-based authentication in the handlers.
- **UI Flow Consistency**: Modified the Electron app to load the root URL (/) instead of directly loading the login page (/login), providing a consistent experience between web and desktop versions.
- **Shared Database Configuration**: Enhanced the application to use a shared database approach that synchronizes data between the Docker container, standalone app, and Electron versions, with proper backup mechanisms.

## Project Structure

```
.
├── db/                 # Database related code
├── electron-app/       # Electron desktop application source and build info
├── static/            # Static assets (CSS, images)
├── templates/         # HTML templates
├── utils/             # Utility functions
└── src/               # Go backend source code
```

## Getting Started

### Prerequisites

- Go 1.16 or higher
- Node.js and npm (for Electron app)
- Docker (optional)
- Windows users: WSL (Windows Subsystem for Linux) for Docker and shell scripts

### Installation

1. Clone the repository:

```bash
git clone https://github.com/Rvin-zh/intercity_portal.git
cd intercity_portal
```

2. Install Go dependencies:

```bash
go mod download
```

3. Install Electron app dependencies:

```bash
cd electron-app
npm install
```

## Running on Windows

Windows users should use WSL (Windows Subsystem for Linux) to run the shell scripts and Docker:

1. Install WSL following [Microsoft's official instructions](https://docs.microsoft.com/en-us/windows/wsl/install)

2. In WSL, navigate to your project directory:

   ```bash
   cd /mnt/c/path/to/project
   ```

3. Run the application using the `run.sh` script:

   ```bash
   ./run.sh
   ```

4. For database setup and configuration:
   ```bash
   ./setup-shared-db.sh
   ```

Alternatively, Windows users can use the Electron desktop version which provides native Windows support:

1. Navigate to the electron-app directory
2. Follow the instructions in the [Electron App README](./electron-app/README.md)

### Running the Application

#### Web Version (Backend)

From the project root:

```bash
go run main.go
```

This will start the backend server. You can access the web interface (currently login page) via your browser, typically at `http://localhost:8080`.

#### Desktop Version (Electron)

For instructions on how to build and run the Electron desktop application, please refer to the specific README located in the `electron-app` directory:

[**Electron App README**](./electron-app/README.md)

#### Docker

From the project root:

```bash
docker-compose up --build
```

**Note for Windows users:** Docker commands should be run inside WSL for best results. Docker Desktop for Windows with WSL integration provides seamless Docker support.

## Development

### Branch Strategy

- `main` - Relatively stable code, potentially deployable
- `develop` - Main development branch where features are merged
- `feature/*` - New features
- `bugfix/*` - Bug fixes
- `release/*` - Release preparation

### Contributing

1. Create a new branch from `develop`
2. Make your changes
3. Submit a pull request against the `develop` branch

## License

This project is licensed under the MIT License.

# Secure Sign In

A secure desktop application for user authentication and management.

## Features

- Secure user authentication
- Password reset with identity verification
- Login history tracking
- User management dashboard
- Cross-platform support (Linux, Windows, macOS)

## Building the Application

### Prerequisites

- Node.js (v16 or later)
- Go (v1.16 or later)
- Git

### Build Scripts

The application provides several build scripts for different platforms:

1. **Automatic Build (Recommended)**

   ```bash
   cd electron-app
   ./builder.sh
   ```

   This script automatically detects your operating system and builds the appropriate package.

2. **Manual Build Options**
   - For Linux (AppImage):
     ```bash
     cd electron-app
     ./build-linux.sh
     ```
   - For Windows (Portable and NSIS):
     ```bash
     cd electron-app
     ./build-windows.sh
     ```
   - For macOS:
     ```bash
     cd electron-app
     ./build-mac.sh
     ```

### Build Output

- Linux: `dist/Secure Sign In-1.0.0.AppImage`
- Windows: `dist/Secure Sign In Setup 1.0.0.exe` and `dist/Secure Sign In-1.0.0-win.exe`
- macOS: `dist/Secure Sign In-1.0.0.dmg`

## Installation

### Linux

1. Make the AppImage executable:
   ```bash
   chmod +x "Secure Sign In-1.0.0.AppImage"
   ```
2. Run the application:
   ```bash
   ./"Secure Sign In-1.0.0.AppImage"
   ```

### Windows

1. Run the NSIS installer (`Secure Sign In Setup 1.0.0.exe`)
2. Follow the installation wizard
3. The application will be installed in your Program Files directory

### macOS

1. Open the DMG file
2. Drag the application to your Applications folder
3. Run from Applications

## Development

### Running in Development Mode

```bash
cd electron-app
npm run dev
```

### Project Structure

- `electron-app/`: Electron frontend application
- `db/`: Database management
- `templates/`: HTML templates
- `handlers/`: Request handlers
- `static/`: Static assets (CSS, JS, images)

## Security Features

- Password hashing with bcrypt
- Secure password reset with identity verification
- Login attempt tracking
- SQLite database with WAL mode
- Cross-site request forgery protection
- Secure session management

## License

MIT License

# SecureSignIn

A secure authentication system with support for SQLite database.

## Features

- User registration and login
- Password reset with security questions
- Dashboard with user management
- Secure password hashing with bcrypt
- Login history tracking

## SQLite Support

This application now uses SQLite for data storage, which provides several benefits:

- Self-contained database (single file)
- Zero configuration required
- Excellent for desktop applications (AppImage, Windows EXE)
- Simplified deployment
- No need for a separate database server

## Running the Application

### Using Docker Compose

The easiest way to run the application is using Docker Compose:

```bash
# Start the application
./run.sh

# Stop the application
./run.sh stop

# View logs
./run.sh logs
```

**Note for Windows users:** Use Windows Subsystem for Linux (WSL) to run these shell scripts. See the [Running on Windows](#running-on-windows) section for details.

### Manual Setup

If you prefer to run the application manually:

1. Install Go 1.18 or later
2. Install SQLite (if not already installed)
3. Clone the repository
4. Run `go mod download` to download dependencies
5. Run `go build -o securesignin .` to build the application
6. Run `./securesignin` to start the application

## Migrating from PostgreSQL to SQLite

If you were previously using PostgreSQL and want to migrate to SQLite, follow these steps:

### Using the Migration Tool

1. Ensure PostgreSQL is still running and accessible
2. Run the migration tool:

```bash
# Build the migration tool
go build -tags migration -o migrate_to_sqlite migrate_postgres_to_sqlite.go

# Run the migration tool (it will connect to PostgreSQL and create a new SQLite database)
./migrate_to_sqlite
```

3. The tool will create a new SQLite database file in the `data` directory
4. Update your application configuration to use SQLite (this is the default now)
5. Start the application using SQLite

### Environment Variables

The application now recognizes the following environment variables:

- `SQLITE_DB_PATH`: Path to the SQLite database file (default: `data/securesignin.db`)

## Database File Location

The SQLite database file is stored in the following locations:

- Docker: `/app/data/securesignin.db` (persisted using Docker volume)
- Standalone: `./data/securesignin.db` (relative to the application directory)

## AppImage and Windows EXE Packaging

SQLite is ideally suited for AppImage and Windows EXE packages since the database is embedded in the application as a file:

1. For AppImage, the SQLite database file can be stored in the user's home directory
2. For Windows EXE, the SQLite database file can be stored in the application directory or user's Documents folder

This eliminates the need for users to install and configure a separate database server.

## Security Considerations

- The SQLite database file contains sensitive information and should be properly secured
- Ensure the database file has appropriate permissions (readable/writable only by the application)
- Consider encrypting the database file for additional security

## License

[License information]

## Data Preservation

This application includes a robust data preservation system that prevents data loss during application updates and rebuilds.

### How It Works

- Before rebuilding the application, user data is automatically backed up
- After the rebuild, data is restored to all possible database locations
- Timestamped backups are created for additional safety
- The system handles different database paths used by:
  - The Electron desktop application
  - The Docker container version
  - The standalone application

### Backup Locations

- Main database: `$HOME/.securesignin/securesignin.db`
- Backup directory: `$HOME/.config/secure-sign-in-app/backups/`

If you need to manually restore a backup:

```bash
# Copy a backup to the main database location
cp ~/.config/secure-sign-in-app/backups/backup-TIMESTAMP.db ~/.securesignin/securesignin.db
```

# Project Setup and Testing

This document provides instructions for setting up the project dependencies and running the tests.

## Prerequisites

- Go (version 1.x or higher)

## Installation

1.  **Clone the repository (if you haven't already):**
    ```bash
    git clone <your-repository-url>
    cd <your-project-directory>
    ```

2.  **Install dependencies:**
    The primary testing dependency is `testify`. To install it and other project dependencies, run:
    ```bash
    go get github.com/stretchr/testify
    ```
    If your project uses Go modules (which is standard), this command will also update your `go.mod` and `go.sum` files.
    You can also run `go mod tidy` to ensure all dependencies are correctly managed:
    ```bash
    go mod tidy
    ```

## Running Tests

### 1. Run All Tests in the Current Package (Verbose)

To run all test functions within the current package (e.g., the package containing `handlers.go` and `handlers_test.go`) and see detailed output for each test (including passing ones), use the following command in your project's root directory:

```bash
go test -v
```

### 2. Run All Tests in the Entire Project (Verbose)

To run all tests in all packages within your project (if you have multiple packages with tests), use:

```bash
go test -v ./...
```

### 3. Run a Single Specific Test Function (Verbose)

To run a single, specific test function (e.g., `TestGenerateResetToken` from `handlers_test.go`), use the `-run` flag followed by a regular expression that matches the test function's name. The `^` and `$` anchors ensure an exact match.

Example for running only `TestGenerateResetToken`:

```bash
go test -v -run ^TestGenerateResetToken$
```

Example for running only `TestStoreAndValidateResetToken`:

```bash
go test -v -run ^TestStoreAndValidateResetToken$
```

**Note:**

*   Ensure you are in the root directory of your Go project when running these commands.
*   The `-v` flag provides verbose output, showing the status of each test as it runs.

@echo off
setlocal enabledelayedexpansion

REM Windows Build Script for Secure Sign In Application
REM This script automatically installs dependencies and builds the application

echo Starting Windows build process...

REM Check if running from the electron-app directory
for %%I in (.) do set CURRENT_DIR=%%~nxI
if not "%CURRENT_DIR%"=="electron-app" (
    echo This script must be run from the electron-app directory.
    echo Please cd into the electron-app directory and try again.
    exit /b 1
)

REM Function to install Chocolatey package manager
:install_chocolatey
echo Checking for Chocolatey package manager...
where choco >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Installing Chocolatey package manager...
    @powershell -NoProfile -ExecutionPolicy Bypass -Command "[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))"
    if %ERRORLEVEL% neq 0 (
        echo Failed to install Chocolatey. Please install it manually.
        exit /b 1
    )
    REM Refresh environment variables to use choco
    call RefreshEnv.cmd
) else (
    echo Chocolatey is already installed.
)

REM Check for Node.js and install if needed
where node >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Node.js not found. Installing...
    choco install nodejs -y
    if %ERRORLEVEL% neq 0 (
        echo Failed to install Node.js. Please install it manually.
        exit /b 1
    )
    REM Refresh environment variables to use node
    call RefreshEnv.cmd
) else (
    echo Node.js is already installed.
)

REM Check for npm and install if needed
where npm >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo npm not found. It should have been installed with Node.js.
    echo Please check your Node.js installation.
    exit /b 1
)

REM Check for Go and install if needed
where go >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Go not found. Installing...
    choco install golang -y
    if %ERRORLEVEL% neq 0 (
        echo Failed to install Go. Please install it manually.
        exit /b 1
    )
    REM Refresh environment variables to use go
    call RefreshEnv.cmd
) else (
    echo Go is already installed.
)

REM Display versions
echo Node.js version:
node --version
echo npm version:
npm --version
echo Go version:
go version

REM Install Node.js dependencies
echo Installing Node.js dependencies...
call npm install
if %ERRORLEVEL% neq 0 (
    echo Failed to install Node.js dependencies.
    exit /b 1
)
echo Dependencies installed successfully.

REM Check if the backend build script exists and make it executable
if not exist build-backend.bat (
    echo Creating build-backend.bat script...
    (
        echo @echo off
        echo REM Build the Go backend for Windows
        echo echo Building Go backend for Windows...
        echo cd ..
        echo go build -o main.exe .
        echo if %%ERRORLEVEL%% neq 0 (
        echo     echo Failed to build Go backend.
        echo     exit /b 1
        echo ^)
        echo echo Go backend built successfully.
        echo cd electron-app
    ) > build-backend.bat
)

REM Build the Go backend
echo Building Go backend for Windows...
call build-backend.bat
if %ERRORLEVEL% neq 0 (
    echo Failed to build Go backend.
    exit /b 1
)
echo Go backend built successfully.

REM Set up encryption keys and directories
echo Setting up encryption keys and directories...
cd ..
if not exist keys (
    mkdir keys
    echo Created keys directory.
)

REM Check for existing encryption key - never generate a new one
if not exist keys\encryption.key (
    echo ERROR: No encryption key found at keys\encryption.key
    echo Please ensure you have a valid encryption key before building!
    echo Keys should be 32 bytes (256 bits) in length.
    exit /b 1
) else (
    echo Using existing encryption key.
)

REM Return to electron-app directory
cd electron-app
echo Encryption key verified successfully.

REM Package the application for Windows
echo Packaging the application for Windows...
call npm run package-win
if %ERRORLEVEL% neq 0 (
    echo Failed to package the application.
    exit /b 1
)

echo Application packaged successfully.
echo The packaged application can be found in the dist/ directory.

REM List the contents of the dist directory
echo Contents of the dist directory:
dir dist

REM Create data preservation script
echo Creating data preservation script...
if not exist "scripts" mkdir scripts

REM Create the preserve-data.js script
if not exist "scripts\preserve-data.js" (
  echo // Database Preservation Tool > scripts\preserve-data.js
  echo const fs = require('fs'); >> scripts\preserve-data.js
  echo const path = require('path'); >> scripts\preserve-data.js
  echo const os = require('os'); >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Define paths >> scripts\preserve-data.js
  echo const HOME_DIR = os.homedir(); >> scripts\preserve-data.js
  echo const DB_PATH = path.join(HOME_DIR, '.securesignin', 'securesignin.db'); >> scripts\preserve-data.js
  echo const BACKUP_DIR = path.join(HOME_DIR, '.config', 'secure-sign-in-app', 'backups'); >> scripts\preserve-data.js
  echo const TEMP_BACKUP_PATH = path.join(BACKUP_DIR, 'pre-build-backup.db'); >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Process arguments >> scripts\preserve-data.js
  echo const args = process.argv.slice(2); >> scripts\preserve-data.js
  echo const operation = args[0] ^|^| 'backup'; // Default to backup >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Ensure backup directory exists >> scripts\preserve-data.js
  echo function ensureBackupDirExists() { >> scripts\preserve-data.js
  echo   if (!fs.existsSync(BACKUP_DIR)) { >> scripts\preserve-data.js
  echo     fs.mkdirSync(BACKUP_DIR, { recursive: true }); >> scripts\preserve-data.js
  echo     console.log(`Created backup directory: ${BACKUP_DIR}`); >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Backup the database before build >> scripts\preserve-data.js
  echo function backupDatabase() { >> scripts\preserve-data.js
  echo   if (!fs.existsSync(DB_PATH)) { >> scripts\preserve-data.js
  echo     console.log(`No database found at ${DB_PATH}. Nothing to backup.`); >> scripts\preserve-data.js
  echo     return false; >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo   ensureBackupDirExists(); >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo   try { >> scripts\preserve-data.js
  echo     fs.copyFileSync(DB_PATH, TEMP_BACKUP_PATH); >> scripts\preserve-data.js
  echo     console.log(`Successfully backed up database to ${TEMP_BACKUP_PATH}`); >> scripts\preserve-data.js
  echo     return true; >> scripts\preserve-data.js
  echo   } catch (error) { >> scripts\preserve-data.js
  echo     console.error(`Error backing up database: ${error.message}`); >> scripts\preserve-data.js
  echo     return false; >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Restore the database after build >> scripts\preserve-data.js
  echo function restoreDatabase() { >> scripts\preserve-data.js
  echo   if (!fs.existsSync(TEMP_BACKUP_PATH)) { >> scripts\preserve-data.js
  echo     console.log(`No backup found at ${TEMP_BACKUP_PATH}. Nothing to restore.`); >> scripts\preserve-data.js
  echo     return false; >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo   // Ensure the target directory exists >> scripts\preserve-data.js
  echo   const dbDir = path.dirname(DB_PATH); >> scripts\preserve-data.js
  echo   if (!fs.existsSync(dbDir)) { >> scripts\preserve-data.js
  echo     fs.mkdirSync(dbDir, { recursive: true }); >> scripts\preserve-data.js
  echo     console.log(`Created database directory: ${dbDir}`); >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo   try { >> scripts\preserve-data.js
  echo     fs.copyFileSync(TEMP_BACKUP_PATH, DB_PATH); >> scripts\preserve-data.js
  echo     console.log(`Successfully restored database to ${DB_PATH}`); >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo     // Create a single permanent backup file >> scripts\preserve-data.js
  echo     const permanentBackup = path.join(BACKUP_DIR, `current-backup.db`); >> scripts\preserve-data.js
  echo     fs.copyFileSync(TEMP_BACKUP_PATH, permanentBackup); >> scripts\preserve-data.js
  echo     console.log(`Updated permanent backup at ${permanentBackup}`); >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo     return true; >> scripts\preserve-data.js
  echo   } catch (error) { >> scripts\preserve-data.js
  echo     console.error(`Error restoring database: ${error.message}`); >> scripts\preserve-data.js
  echo     return false; >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Main function >> scripts\preserve-data.js
  echo function main() { >> scripts\preserve-data.js
  echo   if (operation === 'backup') { >> scripts\preserve-data.js
  echo     console.log('=== Backing up database before build ==='); >> scripts\preserve-data.js
  echo     backupDatabase(); >> scripts\preserve-data.js
  echo   } else if (operation === 'restore') { >> scripts\preserve-data.js
  echo     console.log('=== Restoring database after build ==='); >> scripts\preserve-data.js
  echo     restoreDatabase(); >> scripts\preserve-data.js
  echo   } else { >> scripts\preserve-data.js
  echo     console.error(`Unknown operation: ${operation}`); >> scripts\preserve-data.js
  echo     console.log('Usage: node preserve-data.js [backup^|restore]'); >> scripts\preserve-data.js
  echo     process.exit(1); >> scripts\preserve-data.js
  echo   } >> scripts\preserve-data.js
  echo } >> scripts\preserve-data.js
  echo. >> scripts\preserve-data.js
  echo // Run the main function >> scripts\preserve-data.js
  echo main(); >> scripts\preserve-data.js
)

REM Backup user data before building
echo Backing up user data before build...
node scripts\preserve-data.js backup

REM Create Windows DB setup script
if not exist "..\scripts" mkdir ..\scripts

if not exist "..\scripts\windows-db-setup.bat" (
  echo @echo off > ..\scripts\windows-db-setup.bat
  echo. >> ..\scripts\windows-db-setup.bat
  echo echo === Secure Sign In Database Setup === >> ..\scripts\windows-db-setup.bat
  echo. >> ..\scripts\windows-db-setup.bat
  echo REM Set paths >> ..\scripts\windows-db-setup.bat
  echo set "APP_CONFIG_DIR=%%USERPROFILE%%\.config\secure-sign-in-app" >> ..\scripts\windows-db-setup.bat
  echo set "USER_HOME_DIR=%%USERPROFILE%%\.securesignin" >> ..\scripts\windows-db-setup.bat
  echo set "SHARED_DATA_DIR=%%USERPROFILE%%\SecureSignIn\data" >> ..\scripts\windows-db-setup.bat
  echo set "DB_PATH=%%SHARED_DATA_DIR%%\securesignin.db" >> ..\scripts\windows-db-setup.bat
  echo set "KEY_PATH=%%USER_HOME_DIR%%\encryption.key" >> ..\scripts\windows-db-setup.bat
  echo. >> ..\scripts\windows-db-setup.bat
  echo echo Creating application directories... >> ..\scripts\windows-db-setup.bat
  echo if not exist "%%APP_CONFIG_DIR%%" mkdir "%%APP_CONFIG_DIR%%" >> ..\scripts\windows-db-setup.bat
  echo if not exist "%%USER_HOME_DIR%%" mkdir "%%USER_HOME_DIR%%" >> ..\scripts\windows-db-setup.bat
  echo if not exist "%%APP_CONFIG_DIR%%\backups" mkdir "%%APP_CONFIG_DIR%%\backups" >> ..\scripts\windows-db-setup.bat
  echo if not exist "%%SHARED_DATA_DIR%%" mkdir "%%SHARED_DATA_DIR%%" >> ..\scripts\windows-db-setup.bat
  echo. >> ..\scripts\windows-db-setup.bat
  echo REM If there's an existing database in the user home dir but not in the shared dir, copy it >> ..\scripts\windows-db-setup.bat
  echo if exist "%%USER_HOME_DIR%%\securesignin.db" if not exist "%%DB_PATH%%" ( >> ..\scripts\windows-db-setup.bat
  echo   echo Found existing database in home directory, moving to shared location... >> ..\scripts\windows-db-setup.bat
  echo   copy "%%USER_HOME_DIR%%\securesignin.db" "%%DB_PATH%%" >> ..\scripts\windows-db-setup.bat
  echo ) >> ..\scripts\windows-db-setup.bat
  echo. >> ..\scripts\windows-db-setup.bat
  echo REM Check for encryption key >> ..\scripts\windows-db-setup.bat
  echo if not exist "%%KEY_PATH%%" ( >> ..\scripts\windows-db-setup.bat
  echo   echo Creating encryption key placeholder... >> ..\scripts\windows-db-setup.bat
  echo   REM Generate a random encryption key >> ..\scripts\windows-db-setup.bat
  echo   powershell -Command "$bytes = New-Object Byte[] 32; (New-Object Random).NextBytes($bytes); [IO.File]::WriteAllBytes('%%KEY_PATH%%', $bytes)" >> ..\scripts\windows-db-setup.bat
  echo   if errorlevel 1 ( >> ..\scripts\windows-db-setup.bat
  echo     echo Failed to create encryption key. Please check permissions. >> ..\scripts\windows-db-setup.bat
  echo     exit /b 1 >> ..\scripts\windows-db-setup.bat
  echo   ) >> ..\scripts\windows-db-setup.bat
  echo ) >> ..\scripts\windows-db-setup.bat
  echo. >> ..\scripts\windows-db-setup.bat
  echo echo Database setup complete. Your database will be stored at: %%DB_PATH%% >> ..\scripts\windows-db-setup.bat
  echo echo This shared database location will be used by both Docker and Electron applications. >> ..\scripts\windows-db-setup.bat
)

REM Create Windows run script
if not exist "..\scripts\run-app.bat" (
  echo @echo off > ..\scripts\run-app.bat
  echo. >> ..\scripts\run-app.bat
  echo REM Run script for Secure Sign In application >> ..\scripts\run-app.bat
  echo. >> ..\scripts\run-app.bat
  echo REM Set correct database path >> ..\scripts\run-app.bat
  echo set "SQLITE_DB_PATH=%%USERPROFILE%%\SecureSignIn\data\securesignin.db" >> ..\scripts\run-app.bat
  echo. >> ..\scripts\run-app.bat
  echo REM Run database setup script if it exists >> ..\scripts\run-app.bat
  echo if exist "scripts\windows-db-setup.bat" ( >> ..\scripts\run-app.bat
  echo   call scripts\windows-db-setup.bat >> ..\scripts\run-app.bat
  echo ) >> ..\scripts\run-app.bat
  echo. >> ..\scripts\run-app.bat
  echo REM Run the application >> ..\scripts\run-app.bat
  echo echo Starting Secure Sign In application... >> ..\scripts\run-app.bat
  echo if exist "Secure Sign In Setup 1.0.0.exe" ( >> ..\scripts\run-app.bat
  echo   start "" "Secure Sign In Setup 1.0.0.exe" >> ..\scripts\run-app.bat
  echo ) else if exist "Secure Sign In-1.0.0-win.exe" ( >> ..\scripts\run-app.bat
  echo   start "" "Secure Sign In-1.0.0-win.exe" >> ..\scripts\run-app.bat
  echo ) else ( >> ..\scripts\run-app.bat
  echo   echo Error: Application not found. >> ..\scripts\run-app.bat
  echo   echo Please ensure you're running this script from the application directory. >> ..\scripts\run-app.bat
  echo   exit /b 1 >> ..\scripts\run-app.bat
  echo ) >> ..\scripts\run-app.bat
)

echo Windows scripts created successfully.

REM Restore user data after building
echo Restoring user data after build...
node scripts\preserve-data.js restore

echo Build process completed successfully!
echo Please use run-app.bat to start the application with proper database configuration.
echo Your data has been preserved through the build process.

echo Press any key to exit...
pause > nul 
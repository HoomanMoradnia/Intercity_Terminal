@echo off
setlocal enabledelayedexpansion

echo === Setting up shared database environment for Secure Sign In ===

REM Define paths
set "HOME_DIR=%USERPROFILE%"
set "SHARED_DATA_DIR=%HOME_DIR%\.SecureSignIn\data"
set "ELECTRON_DATA_DIR=%HOME_DIR%\.securesignin"
set "ELECTRON_DB_PATH=%ELECTRON_DATA_DIR%\securesignin.db"
set "SHARED_DB_PATH=%SHARED_DATA_DIR%\securesignin.db"
set "APP_CONFIG_DIR=%HOME_DIR%\.config\secure-sign-in-app"
set "BACKUP_DIR=%APP_CONFIG_DIR%\backups"
set "KEY_PATH=%ELECTRON_DATA_DIR%\encryption.key"

REM Create all necessary directories
echo Creating application directories...
if not exist "%APP_CONFIG_DIR%" mkdir "%APP_CONFIG_DIR%"
if not exist "%ELECTRON_DATA_DIR%" mkdir "%ELECTRON_DATA_DIR%"
if not exist "%BACKUP_DIR%" mkdir "%BACKUP_DIR%"
if not exist "%SHARED_DATA_DIR%" mkdir "%SHARED_DATA_DIR%"

REM If there's an existing database in the Electron directory but not in the shared directory, copy it
if exist "%ELECTRON_DB_PATH%" if not exist "%SHARED_DB_PATH%" (
  echo Found existing database in Electron directory, copying to shared location...
  copy "%ELECTRON_DB_PATH%" "%SHARED_DB_PATH%"
  echo Creating backup of original database...
  copy "%ELECTRON_DB_PATH%" "%BACKUP_DIR%\original-electron-db.bak"
  echo The original database is preserved at: %BACKUP_DIR%\original-electron-db.bak
)

REM Ensure encryption key exists
if not exist "%KEY_PATH%" (
  echo No encryption key found, creating a new one...
  powershell -Command "$bytes = New-Object Byte[] 32; (New-Object Random).NextBytes($bytes); [IO.File]::WriteAllBytes('%KEY_PATH%', $bytes)"
  if errorlevel 1 (
    echo Failed to create encryption key. Please check permissions.
    exit /b 1
  )
)

echo Creating a copy from Electron directory to shared database...
if exist "%ELECTRON_DB_PATH%" (
  echo Moving existing database...
  move "%ELECTRON_DB_PATH%" "%ELECTRON_DB_PATH%.bak"
  echo Backed up existing database to %ELECTRON_DB_PATH%.bak
)

REM Create a symbolic link or copy file for Windows
echo Attempting to create symbolic link...
if exist "%SHARED_DB_PATH%" (
  if exist "%ELECTRON_DB_PATH%" (
    del "%ELECTRON_DB_PATH%"
  )
  REM Try to create symlink (requires admin privileges)
  mklink "%ELECTRON_DB_PATH%" "%SHARED_DB_PATH%" > nul 2>&1
  if errorlevel 1 (
    echo Symbolic link creation failed (requires admin). Creating a file copy instead...
    copy "%SHARED_DB_PATH%" "%ELECTRON_DB_PATH%"
    echo WARNING: This is a copy, not a link. Changes will not be synchronized automatically.
    echo Please run this script as administrator to create proper symbolic links.
  ) else (
    echo Symbolic link created successfully.
  )
)

echo.
echo ====== Setup Complete ======
echo Shared database location: %SHARED_DB_PATH%
echo This database will be used by both Docker and Electron applications.
echo.
echo To run the Docker container with this setup:
echo   docker-compose up -d
echo.
echo To run the Electron app with this setup:
echo   cd electron-app
echo   npm start
echo ==============================

pause 
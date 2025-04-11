@echo off
setlocal enabledelayedexpansion

echo === Secure Sign In Database Setup ===

:: Set paths
set "APP_CONFIG_DIR=%USERPROFILE%\.config\secure-sign-in-app"
set "USER_HOME_DIR=%USERPROFILE%\.securesignin"
set "DB_PATH=%USER_HOME_DIR%\securesignin.db"
set "KEY_PATH=%USER_HOME_DIR%\encryption.key"

:: Create all necessary directories
echo Creating application directories...
if not exist "%APP_CONFIG_DIR%" mkdir "%APP_CONFIG_DIR%"
if not exist "%USER_HOME_DIR%" mkdir "%USER_HOME_DIR%"
if not exist "%APP_CONFIG_DIR%\backups" mkdir "%APP_CONFIG_DIR%\backups"

:: Check for existing encryption key
if not exist "%KEY_PATH%" (
  echo No encryption key found, creating placeholder for app to use
  certutil -f -encodehex NUL "%KEY_PATH%" 32 >nul 2>&1
  if errorlevel 1 (
    echo Failed to create key file. Please run as administrator.
    exit /b 1
  )
)

echo Database setup complete. Your database will be stored at: %DB_PATH%

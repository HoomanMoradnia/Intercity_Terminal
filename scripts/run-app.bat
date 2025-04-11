@echo off
:: Run script for Secure Sign In application

:: Set correct database path
set "SQLITE_DB_PATH=%USERPROFILE%\.securesignin\securesignin.db"

:: Run database setup script if it exists
if exist ".\scripts\windows-db-setup.bat" (
  call ".\scripts\windows-db-setup.bat"
)

:: Run the application
set "APP_PATH=Secure Sign In.exe"
if exist "%APP_PATH%" (
  echo Starting Secure Sign In application...
  start "" "%APP_PATH%"
) else (
  echo Error: Application not found at %APP_PATH%
  echo Please ensure you're running this script from the application directory.
  exit /b 1
)

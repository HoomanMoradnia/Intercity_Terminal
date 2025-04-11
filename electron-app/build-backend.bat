@echo off
setlocal enabledelayedexpansion

REM Script to build the Go backend for Windows

echo Project root directory: %~dp0..
cd %~dp0..

REM Build the Go backend for Windows
echo Building Go backend for Windows...
set "GOOS=windows"
set "GOARCH=amd64"
go build -o main.exe .

REM Create database setup script for Windows
echo Creating database setup scripts for Windows...

mkdir scripts 2>nul

REM Create Windows setup script
echo @echo off > scripts\windows-db-setup.bat
echo setlocal enabledelayedexpansion >> scripts\windows-db-setup.bat
echo. >> scripts\windows-db-setup.bat
echo echo === Secure Sign In Database Setup === >> scripts\windows-db-setup.bat
echo. >> scripts\windows-db-setup.bat
echo REM Set paths >> scripts\windows-db-setup.bat
echo set "APP_CONFIG_DIR=%%USERPROFILE%%\.config\secure-sign-in-app" >> scripts\windows-db-setup.bat
echo set "USER_HOME_DIR=%%USERPROFILE%%\.securesignin" >> scripts\windows-db-setup.bat
echo set "DB_PATH=%%USER_HOME_DIR%%\securesignin.db" >> scripts\windows-db-setup.bat
echo set "KEY_PATH=%%USER_HOME_DIR%%\encryption.key" >> scripts\windows-db-setup.bat
echo. >> scripts\windows-db-setup.bat
echo REM Create all necessary directories >> scripts\windows-db-setup.bat
echo echo Creating application directories... >> scripts\windows-db-setup.bat
echo if not exist "%%APP_CONFIG_DIR%%" mkdir "%%APP_CONFIG_DIR%%" >> scripts\windows-db-setup.bat
echo if not exist "%%USER_HOME_DIR%%" mkdir "%%USER_HOME_DIR%%" >> scripts\windows-db-setup.bat
echo if not exist "%%APP_CONFIG_DIR%%\backups" mkdir "%%APP_CONFIG_DIR%%\backups" >> scripts\windows-db-setup.bat
echo. >> scripts\windows-db-setup.bat
echo REM Check for existing encryption key >> scripts\windows-db-setup.bat
echo if not exist "%%KEY_PATH%%" ( >> scripts\windows-db-setup.bat
echo   echo No encryption key found, creating placeholder for app to use >> scripts\windows-db-setup.bat
echo   certutil -f -encodehex NUL "%%KEY_PATH%%" 32 ^>nul 2^>^&1 >> scripts\windows-db-setup.bat
echo   if errorlevel 1 ( >> scripts\windows-db-setup.bat
echo     echo Failed to create key file. Please run as administrator. >> scripts\windows-db-setup.bat
echo     exit /b 1 >> scripts\windows-db-setup.bat
echo   ) >> scripts\windows-db-setup.bat
echo ) >> scripts\windows-db-setup.bat
echo. >> scripts\windows-db-setup.bat
echo echo Database setup complete. Your database will be stored at: %%DB_PATH%% >> scripts\windows-db-setup.bat

REM Create Windows run script
echo @echo off > scripts\run-app.bat
echo REM Run script for Secure Sign In application >> scripts\run-app.bat
echo. >> scripts\run-app.bat
echo REM Set correct database path >> scripts\run-app.bat
echo set "SQLITE_DB_PATH=%%USERPROFILE%%\.securesignin\securesignin.db" >> scripts\run-app.bat
echo. >> scripts\run-app.bat
echo REM Run database setup script if it exists >> scripts\run-app.bat
echo if exist ".\scripts\windows-db-setup.bat" ( >> scripts\run-app.bat
echo   call ".\scripts\windows-db-setup.bat" >> scripts\run-app.bat
echo ) >> scripts\run-app.bat
echo. >> scripts\run-app.bat
echo REM Run the application >> scripts\run-app.bat
echo set "APP_PATH=Secure Sign In.exe" >> scripts\run-app.bat
echo if exist "%%APP_PATH%%" ( >> scripts\run-app.bat
echo   echo Starting Secure Sign In application... >> scripts\run-app.bat
echo   start "" "%%APP_PATH%%" >> scripts\run-app.bat
echo ) else ( >> scripts\run-app.bat
echo   echo Error: Application not found at %%APP_PATH%% >> scripts\run-app.bat
echo   echo Please ensure you're running this script from the application directory. >> scripts\run-app.bat
echo   exit /b 1 >> scripts\run-app.bat
echo ) >> scripts\run-app.bat

echo Backend build and scripts created successfully! 
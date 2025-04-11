# Secure Sign In - Installation Guide

This guide will help you install and run the Secure Sign In Desktop Application on your computer.

## Linux Installation

### Method 1: AppImage (Recommended for most users)

1. Download the `.AppImage` file from the releases page
2. Make it executable:
   ```bash
   chmod +x Secure_Sign_In-*.AppImage
   ```
3. Double-click the AppImage file or run it from terminal:
   ```bash
   ./Secure_Sign_In-*.AppImage
   ```

### Method 2: Debian Package (Ubuntu, Debian, and derivatives)

1. Download the `.deb` file from the releases page
2. Install using dpkg:
   ```bash
   sudo dpkg -i secure-sign-in-app_*.deb
   ```
   Or double-click the file to open with your system's package manager
3. Launch the application from your applications menu or run:
   ```bash
   secure-sign-in-app
   ```

## Windows Installation

### Method 1: Portable Version (No installation required)

1. Download the portable ZIP file from the releases page
2. Extract the ZIP file to any location on your computer
3. Navigate to the extracted folder
4. Double-click `Secure Sign In.exe` to run the application

### Method 2: Installer

1. Download the installer (`.exe`) from the releases page
2. Double-click the installer file
3. Follow the installation prompts
4. Launch the application from your Start menu or desktop shortcut

## Troubleshooting

### Application Doesn't Start

- **Error message appears**: Follow any instructions in the error message
- **Nothing happens when launching**: Try running from a terminal/command prompt to see error messages:
  - Linux: `./Secure_Sign_In-*.AppImage --verbose`
  - Windows: Open Command Prompt, navigate to the application directory, and run `"Secure Sign In.exe"`

### Missing Dependencies (Linux)

If you see errors about missing libraries, try installing these common dependencies:

```bash
sudo apt-get update
sudo apt-get install libgtk-3-0 libnotify4 libnss3 libxss1 libxtst6 xdg-utils libatspi2.0-0 libuuid1 libsecret-1-0
```

### Security Warnings (Windows)

- **SmartScreen warning**: Click "More info" and then "Run anyway"
- **Antivirus blocking**: Add an exception in your antivirus software

## Uninstalling

### Linux

For AppImage:

- Simply delete the AppImage file

For Debian package:

```bash
sudo apt-get remove secure-sign-in-app
```

### Windows

For portable version:

- Delete the application folder

For installed version:

- Use Windows Settings > Apps > Secure Sign In > Uninstall
- Or use Control Panel > Programs > Uninstall a program

# Packaging and Distribution Guide

This guide explains how to build, package, and distribute the Secure Sign In Desktop Application for both Linux and Windows.

## Prerequisites

- Node.js (v14 or higher)
- npm (v6 or higher)
- Go (v1.16 or higher)
- For Windows builds on Linux: Wine (v1.6 or higher)

If building on Windows, you'll need:

- Windows 10 or higher
- Git Bash or similar shell environment
- Go installed and configured

## Step 1: Install Dependencies

Install all required Node.js dependencies:

```bash
cd electron-app
npm install
```

## Step 2: Build the Backend

Build the Go backend for both Linux and Windows platforms:

```bash
# Make the build script executable (skip this on Windows)
chmod +x build-backend.sh

# Run the build script
./build-backend.sh  # On Windows, you might need to use "sh build-backend.sh"
```

This will create:

- `main` (Linux executable)
- `main.exe` (Windows executable)

These executables will be included in the packaged application.

## Step 3: Package the Application

### For Linux Only

```bash
npm run package-linux
```

This produces:

- An AppImage (`.AppImage`) - Portable, no installation required
- A Debian package (`.deb`) - For Ubuntu, Debian, and derivative distributions

The packaged applications will be in the `dist` directory.

### For Windows Only

```bash
npm run package-win
```

This produces:

- A portable executable (`.exe`) - Run directly, no installation required
- An NSIS installer (`.exe`) - Standard Windows installer

The packaged applications will be in the `dist` directory.

### For Both Platforms

```bash
npm run package-all
```

This builds packages for both Linux and Windows platforms.

## Step 4: Testing the Packaged Application

### Testing on Linux

For the AppImage:

```bash
chmod +x dist/Secure\ Sign\ In-*.AppImage
./dist/Secure\ Sign\ In-*.AppImage
```

For the Debian package:

```bash
sudo dpkg -i dist/secure-sign-in-app_*.deb
# Or use a graphical package manager to install it
```

### Testing on Windows

For the portable executable:

- Navigate to `dist/win-unpacked` directory
- Double-click `Secure Sign In.exe`

For the installer:

- Double-click the installer in the `dist` directory
- Follow the installation prompts

## Distribution

### Linux Distribution

- **AppImage**: Share the `.AppImage` file directly. Users make it executable and run it.
- **Debian Package**: Share the `.deb` file. Users install it using `dpkg` or a package manager.

### Windows Distribution

- **Portable**: Share the contents of the `win-unpacked` directory or create a zip archive of it.
- **Installer**: Share the NSIS installer. Users run it to install the application.

## Adding Auto-Update (Optional)

For auto-updates, you would need:

1. A server to host update files
2. Modifications to `electron-builder` configuration in `package.json`
3. Implementation of update checking in the main process

The basic configuration in `package.json` would look like:

```json
{
  "build": {
    "publish": [
      {
        "provider": "generic",
        "url": "https://your-update-server.com/updates"
      }
    ]
  }
}
```

And you would need to implement update checking in `main.js`.

## Troubleshooting

### Common Issues

1. **Missing dependencies when building the Go backend**:

   ```
   go: cannot find module for path ...
   ```

   Solution: Run `go mod tidy` in the root directory before building.

2. **Wine issues when building for Windows on Linux**:

   ```
   wine: command not found
   ```

   Solution: Install Wine with `sudo apt-get install wine-stable` (Ubuntu/Debian) or equivalent for your distribution.

3. **Application starts but shows a blank window**:
   The backend might not be starting correctly. Check the logs in the terminal or use `DEBUG=1 npm start` for more detailed logs.

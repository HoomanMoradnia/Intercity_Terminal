const { app, BrowserWindow, dialog, shell } = require('electron');
const path = require('path');
const { spawn, exec } = require('child_process');
const os = require('os');
const fs = require('fs');
const http = require('http');

// Keep a global reference of the window object to avoid garbage collection
let mainWindow;
let backendProcess = null;
let portCheckInterval = null;
const BACKEND_PORT = 8080;

// Get path to backend executable
function getBackendPath() {
  const isPackaged = app.isPackaged;
  let backendExecutablePath;

  if (isPackaged) {
    // In packaged app (AppImage, etc.), resourcesPath points to the 'resources' dir
    // The backend files were copied into a 'backend' subdirectory within 'resources'
    const backendDir = path.join(process.resourcesPath, 'backend');
    const exeName = os.platform() === 'win32' ? 'main.exe' : 'main';
    backendExecutablePath = path.join(backendDir, exeName);
    console.log(`Packaged app detected. Resources path: ${process.resourcesPath}`);
  } else {
    // In development, backend is in the project root (two levels up from src)
    const baseDir = path.join(__dirname, '..', '..');
    const exeName = os.platform() === 'win32' ? 'main.exe' : 'main';
    backendExecutablePath = path.join(baseDir, exeName);
    console.log(`Development mode detected. Project path: ${baseDir}`);
  }

  console.log(`Resolved backend executable path: ${backendExecutablePath}`);
  return backendExecutablePath;
}

// Check if backend is running
async function checkBackendReachable(url) {
  return new Promise((resolve) => {
    const req = http.get(url, (res) => {
      res.resume();
      resolve(res.statusCode === 200);
    }).on('error', () => {
      resolve(false);
    });
    req.setTimeout(1000, () => {
      req.abort();
      resolve(false);
    });
  });
}

// Force kill any process using port 8080
async function killProcessOnPort(port) {
  return new Promise((resolve) => {
    console.log(`Attempting to kill any process using port ${port}...`);
    
    if (os.platform() === 'win32') {
      // Windows command to find and kill process on port
      exec(`for /f "tokens=5" %a in ('netstat -aon ^| find ":${port}" ^| find "LISTENING"') do taskkill /F /PID %a`, (error) => {
        if (error) {
          console.log(`No process found on port ${port} or failed to kill: ${error.message}`);
        } else {
          console.log(`Successfully killed process on port ${port}`);
        }
        resolve();
      });
    } else {
      // Linux/macOS command
      exec(`lsof -i :${port} -t | xargs -r kill -9`, (error) => {
        if (error) {
          console.log(`No process found on port ${port} or failed to kill: ${error.message}`);
        } else {
          console.log(`Successfully killed process on port ${port}`);
        }
        resolve();
      });
    }
  });
}

// Start the backend server
async function startBackend() {
  // First, ensure no process is already using our port
  await killProcessOnPort(BACKEND_PORT);
  
  const backendPath = getBackendPath();
  const backendDirectory = path.dirname(backendPath);
  
  // Get user data paths
  const appDataDir = app.getPath('userData');
  const homeDir = app.getPath('home');
  const secureSignInDir = path.join(homeDir, '.securesignin');
  
  // Set database path - prioritize environment variable, then user .securesignin dir
  const sqliteDbPath = process.env.SQLITE_DB_PATH || path.join(secureSignInDir, 'securesignin.db');
  
  const keyDirectory = path.join(backendDirectory, 'keys');
  const keyFilePath = path.join(keyDirectory, 'encryption.key');

  console.log(`Backend executable path: ${backendPath}`);
  console.log(`Backend working directory: ${backendDirectory}`);
  console.log(`SQLite database path: ${sqliteDbPath}`);
  console.log(`Key directory path: ${keyDirectory}`);
  console.log(`Encryption key path: ${keyFilePath}`);

  // Ensure all required directories exist
  const requiredDirs = [
    appDataDir,
    secureSignInDir,
    path.dirname(sqliteDbPath),
    keyDirectory
  ];
  
  for (const dir of requiredDirs) {
    if (!fs.existsSync(dir)) {
      console.log(`Creating directory: ${dir}`);
      try {
        fs.mkdirSync(dir, { recursive: true });
        console.log(`Directory created successfully: ${dir}`);
      } catch (error) {
        console.error(`Error creating directory ${dir}: ${error.message}`);
      }
    }
  }

  // Check if encryption key exists
  if (!fs.existsSync(keyFilePath)) {
    const errorMsg = `Encryption key not found at: ${keyFilePath}`;
    console.error(errorMsg);
    
    // Check if we have a key in the user home directory that we can copy
    const homeKeyPath = path.join(secureSignInDir, 'encryption.key');
    if (fs.existsSync(homeKeyPath)) {
      console.log(`Found encryption key at ${homeKeyPath}, copying to ${keyFilePath}`);
      try {
        fs.copyFileSync(homeKeyPath, keyFilePath);
        console.log('Encryption key copied successfully.');
      } catch (copyError) {
        console.error(`Error copying key: ${copyError.message}`);
        dialog.showErrorBox('Encryption Key Error', 
          `${errorMsg}\n\nAttempted to copy from ${homeKeyPath} but failed: ${copyError.message}`);
        app.quit();
        return false;
      }
    } else {
      dialog.showErrorBox('Encryption Key Error', 
        `${errorMsg}\n\nPlease ensure the encryption key file exists at 'keys/encryption.key' in the application resources.`);
      app.quit();
      return false;
    }
  }

  if (!fs.existsSync(backendPath)) {
    const errorMsg = `Backend executable not found at: ${backendPath}`;
    console.error(errorMsg);
    dialog.showErrorBox('Backend Error', errorMsg + 
      `\n\nWindows troubleshooting:\n1. Check if antivirus is blocking the executable\n2. Ensure you have proper permissions\n3. Try running as administrator`);
    app.quit();
    return false; // Indicate failure
  }

  // Check backend executable permissions
  try {
    fs.accessSync(backendPath, fs.constants.X_OK);
    console.log('Backend executable has execute permissions.');
  } catch (error) {
    console.warn(`Backend executable doesn't have execute permissions. Attempting to fix...`);
    if (os.platform() !== 'win32') { // Windows doesn't use the same permission model
      try {
        fs.chmodSync(backendPath, 0o755); // Set execute permissions
        console.log('Execute permissions set on backend executable.');
      } catch (chmodErr) {
        console.error(`Failed to set execute permissions: ${chmodErr.message}`);
      }
    }
  }

  // Double-check if backend is already running
  if (await checkBackendReachable(`http://localhost:${BACKEND_PORT}/health`)) {
    console.log('Backend appears to be running already, but we just killed port processes. This should not happen.');
    await killProcessOnPort(BACKEND_PORT); // Try again
    await new Promise(resolve => setTimeout(resolve, 1000)); // Wait a moment
  }

  try {
    // For Windows, log the exact command we're about to execute
    if (os.platform() === 'win32') {
      console.log(`Starting Windows backend with: "${backendPath}" in directory: ${backendDirectory}`);
    }
    
    backendProcess = spawn(backendPath, [], {
      cwd: backendDirectory,
      // Do not detach the process - this was causing issues with process termination
      detached: false,
      shell: os.platform() === 'win32',
      env: {
        ...process.env, // Inherit environment
        USE_SQLITE: '1', // Tell backend to use SQLite
        SQLITE_DB_PATH: sqliteDbPath, // Provide path for the db file
        KEY_DIR: keyDirectory, // Specify key directory path
        KEY_FILE: keyFilePath // Direct path to the encryption key
      }
    });

    // Track the process ID for clean termination
    console.log(`Backend process started with PID: ${backendProcess.pid}`);

    backendProcess.stdout.on('data', (data) => {
      console.log(`Backend stdout: ${data.toString().trim()}`);
    });
    backendProcess.stderr.on('data', (data) => {
      console.error(`Backend stderr: ${data.toString().trim()}`);
    });
    backendProcess.on('error', (err) => {
      const errorMsg = `Failed to start backend process: ${err}`;
      console.error(errorMsg);
      let detailedMsg = errorMsg;
      
      if (os.platform() === 'win32') {
        detailedMsg += `\n\nWindows troubleshooting:\n- Make sure no antivirus is blocking the app\n- Try running as administrator\n- Check Windows Defender settings\n- Verify the backend executable exists at: ${backendPath}`;
      }
      
      dialog.showErrorBox('Backend Error', detailedMsg);
      if (!app.isQuitting) app.quit();
    });
    backendProcess.on('close', (code) => {
      console.log(`Backend process exited with code ${code}`);
      backendProcess = null;
      if (code !== 0 && !app.isQuitting) {
        let errorMsg = `The backend server stopped unexpectedly (code: ${code}).`;
        
        if (os.platform() === 'win32') {
          errorMsg += `\n\nWindows troubleshooting:\n- Check if Windows Firewall is blocking network access\n- Verify SQLite database path is accessible: ${sqliteDbPath}\n- Verify encryption key exists at: ${keyFilePath}\n- Run the app as administrator`;
        }
        
        dialog.showErrorBox('Backend Error', errorMsg);
        app.quit();
      }
    });

    console.log('Backend process started, waiting for it to become reachable...');
    return true; // Indicate success (started)

  } catch (error) {
    console.error('Error spawning backend process:', error);
    let errorMsg = `Error starting the backend server: ${error.message}`;
    
    if (os.platform() === 'win32') {
      errorMsg += `\n\nWindows troubleshooting:\n- Verify all DLLs are present in resources/backend\n- Check Windows permissions\n- Try running as administrator`;
    }
    
    dialog.showErrorBox('Backend Error', errorMsg);
    if (!app.isQuitting) app.quit();
    return false; // Indicate failure
  }
}

// Create the browser window
function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true
    },
    // Optional: Define icon path relative to build output
    // icon: path.join(__dirname, '..\/assets\/icon.png') // Adjust if you add an icon
  });

  mainWindow.loadURL(`http://localhost:${BACKEND_PORT}/login`);

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith('http')) {
      shell.openExternal(url);
      return { action: 'deny' };
    }
    return { action: 'allow' };
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// Wait for backend to be ready
async function waitForBackend() {
  console.log('Checking if backend is reachable...');
  const healthUrl = `http://localhost:${BACKEND_PORT}/health`;
  let backendReady = false;
  let attempts = 0;
  const maxAttempts = 30; // Wait up to 30 seconds

  while (!backendReady && attempts < maxAttempts) {
    attempts++;
    backendReady = await checkBackendReachable(healthUrl);
    if (backendReady) {
      console.log(`Backend is reachable after ${attempts} attempt(s).`);
      return true;
    } else {
      console.log(`Waiting for backend... Attempt ${attempts}/${maxAttempts}`);
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }

  console.error('Backend did not become reachable.');
  dialog.showErrorBox('Backend Error', 'The backend service failed to start in time.');
  return false;
}

// Ensure backend is properly terminated
function terminateBackend() {
  return new Promise(async (resolve) => {
    if (backendProcess) {
      console.log(`Terminating backend process with PID: ${backendProcess.pid}`);
      
      // Try graceful termination first
      if (os.platform() === 'win32') {
        exec(`taskkill /pid ${backendProcess.pid} /T /F`, (error) => {
          if (error) console.error(`Failed to kill Windows process: ${error.message}`);
          backendProcess = null;
          
          // Force kill any remaining process on port 8080
          killProcessOnPort(BACKEND_PORT).then(resolve);
        });
      } else {
        try {
          backendProcess.kill('SIGTERM');
          console.log('SIGTERM sent to backend process');
          
          // Give it a moment to terminate
          setTimeout(() => {
            // If still running, force kill
            if (backendProcess) {
              try {
                backendProcess.kill('SIGKILL');
                console.log('SIGKILL sent to backend process');
              } catch (e) {
                console.error(`Error sending SIGKILL: ${e.message}`);
              }
            }
            
            // Force kill any remaining process on port 8080
            killProcessOnPort(BACKEND_PORT).then(resolve);
          }, 1000);
        } catch (error) {
          console.error(`Error terminating backend: ${error.message}`);
          killProcessOnPort(BACKEND_PORT).then(resolve);
        }
      }
    } else {
      // No known process, but let's make sure the port is free
      killProcessOnPort(BACKEND_PORT).then(resolve);
    }
  });
}

// Electron App Lifecycle
app.whenReady().then(async () => {
  // First, make sure no previous instance is running
  await killProcessOnPort(BACKEND_PORT);
  
  const backendStarted = await startBackend();
  if (!backendStarted) {
    if (!app.isQuitting) app.quit();
    return;
  }

  const ready = await waitForBackend();
  if (ready) {
    createWindow();
  } else {
    if (!app.isQuitting) app.quit();
  }

  app.on('activate', () => {
    if (mainWindow === null) {
      // Ensure backend is ready before creating window on activate
      waitForBackend().then(ready => {
        if (ready) createWindow();
      });
    }
  });
});

app.on('before-quit', async (event) => {
  if (!app.isTerminating) {
    event.preventDefault();
    app.isQuitting = true;
    app.isTerminating = true;
    
    console.log('Application is quitting, cleaning up resources...');
    await terminateBackend();
    
    // Continue with quit after cleanup
    console.log('Cleanup complete, quitting application');
    app.quit();
  }
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('will-quit', async (event) => {
  if (!app.isTerminating) {
    event.preventDefault();
    app.isTerminating = true;
    
    console.log('Final termination check before quit...');
    await terminateBackend();
    
    app.quit();
  }
}); 
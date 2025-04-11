// This method will be called when Electron has finished initialization
app.whenReady().then(async () => {
  // Instead of starting the backend, just create the window
  // The backend is already running via Docker
  createWindow();
  
  app.on('activate', function () {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open
    if (mainWindow === null) createWindow();
  });
}); 

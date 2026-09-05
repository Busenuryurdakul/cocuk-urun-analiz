const { app, BrowserWindow, ipcMain, shell } = require("electron");
const path = require("path");
const crypto = require("crypto");
const os = require("os");

let keytar;
try {
  keytar = require("keytar");
} catch {
  keytar = null;
}

let machineId;
try {
  machineId = require("node-machine-id").machineIdSync;
} catch {
  machineId = null;
}

const WEB_URL = process.env.MIYUNA_WEB_URL || "http://localhost:3000";
const KEYTAR_SERVICE = "miyuna-desktop";
const KEYTAR_ACCOUNT = "refresh-token";

/** @type {BrowserWindow | null} */
let mainWindow = null;
/** @type {string | null} */
let pendingDeepLink = null;

function isAllowedUrl(target) {
  try {
    const allowed = new URL(WEB_URL);
    const next = new URL(target);
    return next.origin === allowed.origin;
  } catch {
    return false;
  }
}

function stableDeviceFingerprint() {
  if (machineId) {
    const raw = machineId(true);
    return crypto.createHash("sha256").update(raw).digest("hex");
  }
  return crypto.createHash("sha256").update(os.hostname() + os.userInfo().username).digest("hex");
}

function registerProtocol() {
  if (process.defaultApp) {
    if (process.argv.length >= 2) {
      app.setAsDefaultProtocolClient("miyuna", process.execPath, [path.resolve(process.argv[1])]);
    }
  } else {
    app.setAsDefaultProtocolClient("miyuna");
  }
}

function handleDeepLink(url) {
  if (!url || !url.startsWith("miyuna://")) return;
  pendingDeepLink = url;
  if (mainWindow) {
    mainWindow.webContents.send("deep-link", url);
    if (mainWindow.isMinimized()) mainWindow.restore();
    mainWindow.focus();
  }
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 840,
    minWidth: 960,
    minHeight: 640,
    show: false,
    backgroundColor: "#14352f",
    title: "Miyuna",
    frame: false,
    titleBarStyle: process.platform === "darwin" ? "hiddenInset" : "hidden",
    trafficLightPosition: { x: 16, y: 10 },
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webviewTag: false,
    },
  });

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (isAllowedUrl(url)) return { action: "allow" };
    void shell.openExternal(url);
    return { action: "deny" };
  });

  mainWindow.webContents.on("will-navigate", (event, url) => {
    if (url.startsWith("miyuna://")) {
      event.preventDefault();
      handleDeepLink(url);
      return;
    }
    if (!isAllowedUrl(url)) {
      event.preventDefault();
      void shell.openExternal(url);
    }
  });

  mainWindow.once("ready-to-show", () => {
    mainWindow?.show();
    if (pendingDeepLink) {
      mainWindow?.webContents.send("deep-link", pendingDeepLink);
    }
  });

  void mainWindow.loadURL(WEB_URL);
}

function setupIPC() {
  ipcMain.handle("device:fingerprint", () => stableDeviceFingerprint());
  ipcMain.handle("app:version", () => app.getVersion());
  ipcMain.handle("token:set", async (_event, token) => {
    if (!keytar) return false;
    await keytar.setPassword(KEYTAR_SERVICE, KEYTAR_ACCOUNT, token);
    return true;
  });
  ipcMain.handle("token:get", async () => {
    if (!keytar) return null;
    return keytar.getPassword(KEYTAR_SERVICE, KEYTAR_ACCOUNT);
  });
  ipcMain.handle("token:clear", async () => {
    if (!keytar) return false;
    return keytar.deletePassword(KEYTAR_SERVICE, KEYTAR_ACCOUNT);
  });
  ipcMain.on("window:minimize", () => mainWindow?.minimize());
  ipcMain.on("window:maximize", () => {
    if (!mainWindow) return;
    if (mainWindow.isMaximized()) mainWindow.unmaximize();
    else mainWindow.maximize();
  });
  ipcMain.on("window:close", () => mainWindow?.close());
}

const gotLock = app.requestSingleInstanceLock();
if (!gotLock) {
  app.quit();
} else {
  app.on("second-instance", (_event, argv) => {
    const deepLink = argv.find((arg) => arg.startsWith("miyuna://"));
    if (deepLink) handleDeepLink(deepLink);
    if (!mainWindow) return;
    if (mainWindow.isMinimized()) mainWindow.restore();
    mainWindow.focus();
  });

  app.whenReady().then(() => {
    registerProtocol();
    setupIPC();
    createWindow();
    app.on("open-url", (event, url) => {
      event.preventDefault();
      handleDeepLink(url);
    });
    app.on("activate", () => {
      if (BrowserWindow.getAllWindows().length === 0) createWindow();
    });
  });
}

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

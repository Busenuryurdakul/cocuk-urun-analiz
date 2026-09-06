const { app, BrowserWindow, ipcMain, shell, session, Menu, nativeImage } = require("electron");
const path = require("path");
const crypto = require("crypto");
const os = require("os");
const fs = require("fs");

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

const WEB_URL = (process.env.MIYUNA_WEB_URL || "http://localhost:3000").replace(/\/+$/, "");
const API_URL = (process.env.MIYUNA_API_URL || "http://localhost:8080").replace(/\/+$/, "");
const KEYTAR_SERVICE = "miyuna-desktop";
const KEYTAR_ACCOUNT = "refresh-token";
const SESSION_PARTITION = "miyuna-live";
const isDev = !app.isPackaged;

/** @type {BrowserWindow | null} */
let mainWindow = null;
/** @type {string | null} */
let pendingDeepLink = null;
/** @type {Electron.Session | null} */
let appSession = null;

function isAllowedUrl(target) {
  try {
    const allowed = new URL(WEB_URL);
    const next = new URL(target);
    return next.origin === allowed.origin;
  } catch {
    return false;
  }
}

function isExternalHttp(target) {
  try {
    const next = new URL(target);
    return next.protocol === "https:" || next.protocol === "http:";
  } catch {
    return false;
  }
}

function stableDeviceFingerprint() {
  if (machineId) {
    const raw = machineId(true);
    return crypto.createHash("sha256").update(raw).digest("hex");
  }
  return crypto.createHash("sha256").update(`${os.hostname()}|${os.userInfo().username}|${os.platform()}`).digest("hex");
}

function resolveIcon() {
  const png = path.join(__dirname, "assets", "icon.png");
  if (fs.existsSync(png)) {
    return nativeImage.createFromPath(png);
  }
  return undefined;
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

async function persistRefreshToken(token) {
  if (!keytar) return false;
  try {
    if (!token) {
      await keytar.deletePassword(KEYTAR_SERVICE, KEYTAR_ACCOUNT);
      return true;
    }
    await keytar.setPassword(KEYTAR_SERVICE, KEYTAR_ACCOUNT, token);
    return true;
  } catch {
    return false;
  }
}

async function readRefreshToken() {
  if (!keytar) return null;
  return keytar.getPassword(KEYTAR_SERVICE, KEYTAR_ACCOUNT);
}

function hardenSession(ses) {
  ses.setPermissionRequestHandler((_webContents, _permission, callback) => {
    callback(false);
  });
  ses.setPermissionCheckHandler(() => false);
  if (typeof ses.setDevicePermissionHandler === "function") {
    ses.setDevicePermissionHandler(() => false);
  }
  ses.on("will-download", (event) => {
    event.preventDefault();
  });
  ses.cookies.on("changed", (_event, cookie, _cause, removed) => {
    if (cookie.name !== "miyuna_refresh") return;
    void persistRefreshToken(removed ? "" : cookie.value);
  });
}

async function rehydrateSession(ses) {
  const token = await readRefreshToken();
  if (!token) return;
  await ses.cookies.set({
    url: API_URL,
    name: "miyuna_refresh",
    value: token,
    path: "/",
    httpOnly: true,
    secure: API_URL.startsWith("https:"),
    sameSite: "lax",
  });
  try {
    const res = await ses.fetch(`${API_URL}/graphql`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query: "mutation { refreshToken }" }),
    });
    const payload = await res.json();
    if (payload.errors?.length) {
      await persistRefreshToken("");
      await ses.cookies.remove(API_URL, "miyuna_refresh");
    }
  } catch {
    // Keep the keychain token if the API is temporarily unreachable.
  }
}

function loadOffline(win) {
  const offline = path.join(__dirname, "offline.html");
  void win.loadFile(offline);
}

function createWindow(ses) {
  const icon = resolveIcon();
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 840,
    minWidth: 960,
    minHeight: 640,
    show: false,
    backgroundColor: "#14352f",
    title: "Miyuna",
    icon,
    frame: false,
    titleBarStyle: process.platform === "darwin" ? "hiddenInset" : "hidden",
    trafficLightPosition: { x: 16, y: 10 },
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      session: ses,
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webviewTag: false,
      spellcheck: false,
      navigateOnDragDrop: false,
    },
  });

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (isAllowedUrl(url)) return { action: "allow" };
    if (isExternalHttp(url)) void shell.openExternal(url);
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
      if (isExternalHttp(url)) void shell.openExternal(url);
    }
  });

  mainWindow.webContents.on("will-attach-webview", (event) => {
    event.preventDefault();
  });

  if (!isDev) {
    mainWindow.webContents.on("devtools-opened", () => {
      mainWindow?.webContents.closeDevTools();
    });
  }

  mainWindow.webContents.on("did-fail-load", (_event, errorCode) => {
    if (errorCode === -3) return;
    loadOffline(mainWindow);
  });

  mainWindow.once("ready-to-show", () => {
    mainWindow?.show();
    if (pendingDeepLink) {
      mainWindow?.webContents.send("deep-link", pendingDeepLink);
    }
  });

  void mainWindow.loadURL(WEB_URL).catch(() => loadOffline(mainWindow));
}

function setupIPC() {
  ipcMain.handle("device:fingerprint", () => stableDeviceFingerprint());
  ipcMain.handle("app:version", () => app.getVersion());
  ipcMain.handle("token:set", async (_event, token) => persistRefreshToken(String(token || "")));
  ipcMain.handle("token:get", async () => readRefreshToken());
  ipcMain.handle("token:clear", async () => persistRefreshToken(""));
  ipcMain.on("window:minimize", () => mainWindow?.minimize());
  ipcMain.on("window:maximize", () => {
    if (!mainWindow) return;
    if (mainWindow.isMaximized()) mainWindow.unmaximize();
    else mainWindow.maximize();
  });
  ipcMain.on("window:close", () => mainWindow?.close());
  ipcMain.on("app:retry", () => {
    if (!mainWindow) return;
    void mainWindow.loadURL(WEB_URL).catch(() => loadOffline(mainWindow));
  });
}

const gotLock = app.requestSingleInstanceLock();
if (!gotLock) {
  app.quit();
} else {
  app.on("second-instance", (_event, argv) => {
    const deepLink = argv.find((arg) => typeof arg === "string" && arg.startsWith("miyuna://"));
    if (deepLink) handleDeepLink(deepLink);
    if (!mainWindow) return;
    if (mainWindow.isMinimized()) mainWindow.restore();
    mainWindow.focus();
  });

  app.whenReady().then(async () => {
    registerProtocol();
    if (process.platform === "win32") {
      Menu.setApplicationMenu(null);
    }
    appSession = session.fromPartition(SESSION_PARTITION);
    hardenSession(appSession);
    setupIPC();
    await rehydrateSession(appSession);
    createWindow(appSession);
    app.on("open-url", (event, url) => {
      event.preventDefault();
      handleDeepLink(url);
    });
    app.on("activate", () => {
      if (BrowserWindow.getAllWindows().length === 0 && appSession) createWindow(appSession);
    });
  });
}

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

app.on("web-contents-created", (_event, contents) => {
  contents.on("will-navigate", (event, url) => {
    if (url.startsWith("miyuna://")) {
      event.preventDefault();
      handleDeepLink(url);
      return;
    }
    if (!isAllowedUrl(url) && contents !== mainWindow?.webContents) {
      event.preventDefault();
    }
  });
});

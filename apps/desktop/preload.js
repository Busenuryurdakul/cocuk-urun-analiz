const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("miyunaDesktop", {
  platform: process.platform,
  getDeviceFingerprint: () => ipcRenderer.invoke("device:fingerprint"),
  getAppVersion: () => ipcRenderer.invoke("app:version"),
  setRefreshToken: (token) => ipcRenderer.invoke("token:set", token),
  getRefreshToken: () => ipcRenderer.invoke("token:get"),
  clearTokens: () => ipcRenderer.invoke("token:clear"),
  onDeepLink: (callback) => {
    ipcRenderer.on("deep-link", (_event, url) => callback(url));
  },
  minimize: () => ipcRenderer.send("window:minimize"),
  maximize: () => ipcRenderer.send("window:maximize"),
  close: () => ipcRenderer.send("window:close"),
});

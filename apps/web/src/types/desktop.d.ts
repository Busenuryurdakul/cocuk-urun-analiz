export {};

type MiyunaDesktopBridge = {
  platform: string;
  getDeviceFingerprint: () => Promise<string>;
  getAppVersion: () => Promise<string> | string;
  setRefreshToken?: (token: string) => Promise<boolean>;
  getRefreshToken?: () => Promise<string | null>;
  clearTokens?: () => Promise<boolean>;
  onDeepLink?: (callback: (url: string) => void) => (() => void) | void;
  retryConnection?: () => void;
  minimize?: () => void;
  maximize?: () => void;
  close?: () => void;
};

declare global {
  interface Window {
    miyunaDesktop?: MiyunaDesktopBridge;
  }
}

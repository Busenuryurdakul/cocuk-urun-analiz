export {};

declare global {
  interface Window {
    miyunaDesktop?: {
      platform: string;
      minimize: () => void;
      maximize: () => void;
      close: () => void;
    };
  }
}

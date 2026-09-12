const STORAGE_KEY = "miyuna_pending_token";

export function savePendingToken(token: string | null | undefined): void {
  if (typeof window === "undefined") return;
  if (!token) {
    window.sessionStorage.removeItem(STORAGE_KEY);
    return;
  }
  window.sessionStorage.setItem(STORAGE_KEY, token);
}

export function readPendingToken(): string | null {
  if (typeof window === "undefined") return null;
  const token = window.sessionStorage.getItem(STORAGE_KEY);
  return token?.trim() ? token : null;
}

export function clearPendingToken(): void {
  savePendingToken(null);
}

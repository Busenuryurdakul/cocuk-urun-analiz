function resolveApiBaseUrl(): string {
  const raw = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").trim().replace(/\/+$/, "");
  return raw.replace(/\/graphql$/i, "");
}

const API_URL = resolveApiBaseUrl();

type GraphQLResponse<T> = {
  data?: T;
  errors?: Array<{ message: string; extensions?: { code?: string } }>;
};

export class GraphQLRequestError extends Error {
  readonly code: string;

  constructor(code: string) {
    super(code);
    this.name = "GraphQLRequestError";
    this.code = code;
  }
}

export function graphqlErrorCode(err: unknown): string | undefined {
  if (err instanceof GraphQLRequestError) {
    return err.code;
  }
  if (err instanceof Error) {
    return err.message;
  }
  return undefined;
}

const AUTH_ERROR_MESSAGES: Record<string, string> = {
  EMAIL_NOT_VERIFIED:
    "E-posta adresiniz henüz doğrulanmadı. Kayıt sırasında gönderilen bağlantıyı açın, ardından tekrar giriş yapın.",
  INVALID_CREDENTIALS: "E-posta veya şifre hatalı.",
  INVALID_TOKEN: "Oturum süresi doldu veya geçersiz. Giriş sayfasından tekrar deneyin.",
  INVALID_CODE: "Doğrulama kodu hatalı veya süresi dolmuş.",
  CHALLENGE_LOCKED: "Çok fazla deneme yapıldı. Lütfen daha sonra tekrar deneyin.",
  DUPLICATE: "Bu e-posta ile zaten bir hesap var.",
  DESKTOP_SESSION_ACTIVE:
    "Bu hesapta başka bir masaüstü oturumu açık. Önce diğer cihazdan çıkış yapın veya o cihazı güvenlik ayarlarından kaldırın.",
  PERSONAL_ORG: "Kişisel çalışma alanına üye davet edilemez. Ekip için önce bir organizasyon oluşturun.",
  ALREADY_MEMBER: "Bu e-posta zaten bu organizasyonun üyesi.",
  INVITATION_PENDING: "Bu e-postaya zaten bekleyen bir davet var.",
  CONSENT_REQUIRED: "Davet için uyumluluk onayı gerekli. Uyumluluk sayfasından onayları verip tekrar deneyin.",
};

function isNetworkFailure(err: unknown): boolean {
  if (err instanceof TypeError) {
    return true;
  }
  const message = err instanceof Error ? err.message : "";
  return (
    message === "Failed to fetch" ||
    message.startsWith("GraphQL HTTP") ||
    message.includes("NetworkError") ||
    message.includes("Load failed")
  );
}

export function authErrorMessage(err: unknown, fallback = "İşlem başarısız"): string {
  const code = graphqlErrorCode(err);
  if (code && AUTH_ERROR_MESSAGES[code]) {
    return AUTH_ERROR_MESSAGES[code];
  }
  if (isNetworkFailure(err)) {
    return "Sunucuya bağlanılamadı. Lütfen biraz sonra tekrar deneyin.";
  }
  return fallback;
}

type MiyunaDesktop = {
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
    miyunaDesktop?: MiyunaDesktop;
  }
}

export function isDesktopClient(): boolean {
  return typeof window !== "undefined" && Boolean(window.miyunaDesktop);
}

export function clientPlatform(): "WEB" | "ELECTRON_WIN" | "ELECTRON_MAC" {
  if (typeof window === "undefined") {
    return "WEB";
  }
  const platform = window.miyunaDesktop?.platform;
  if (platform === "win32") return "ELECTRON_WIN";
  if (platform === "darwin") return "ELECTRON_MAC";
  return "WEB";
}

let desktopFingerprintPromise: Promise<string> | null = null;

export async function deviceFingerprintAsync(): Promise<string> {
  if (typeof window === "undefined") {
    return "server";
  }
  if (window.miyunaDesktop?.getDeviceFingerprint) {
    if (!desktopFingerprintPromise) {
      desktopFingerprintPromise = window.miyunaDesktop.getDeviceFingerprint();
    }
    return desktopFingerprintPromise;
  }
  return deviceFingerprint();
}

export function deviceFingerprint(): string {
  if (typeof window === "undefined") {
    return "server";
  }
  const key = "miyuna_device_fp";
  const existing = window.localStorage.getItem(key);
  if (existing) {
    return existing;
  }
  const fp = crypto.randomUUID();
  window.localStorage.setItem(key, fp);
  return fp;
}

export async function appVersion(): Promise<string | undefined> {
  if (typeof window === "undefined" || !window.miyunaDesktop?.getAppVersion) {
    return undefined;
  }
  return window.miyunaDesktop.getAppVersion();
}

export async function clearDesktopSession(): Promise<void> {
  if (typeof window === "undefined") return;
  await window.miyunaDesktop?.clearTokens?.();
}

export async function graphqlRequest<T>(
  query: string,
  variables?: Record<string, unknown>,
): Promise<T> {
  const res = await fetch(`${API_URL}/graphql`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ query, variables }),
  });

  if (!res.ok) {
    throw new Error(`GraphQL HTTP ${res.status}`);
  }

  const payload = (await res.json()) as GraphQLResponse<T>;
  if (payload.errors?.length && !hasUsableGraphQLData(payload.data)) {
    const code = payload.errors[0].extensions?.code ?? payload.errors[0].message;
    throw new GraphQLRequestError(code);
  }
  if (!payload.data) {
    throw new Error("GraphQL yanıtı boş");
  }
  return payload.data;
}

function hasUsableGraphQLData<T>(data: T | undefined): data is T {
  if (data == null || typeof data !== "object") {
    return false;
  }
  return Object.values(data as Record<string, unknown>).some((value) => value != null && value !== false);
}

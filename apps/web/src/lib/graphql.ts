const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

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
  INVALID_TOKEN: "Bağlantı geçersiz veya süresi dolmuş.",
  CHALLENGE_LOCKED: "Çok fazla deneme yapıldı. Lütfen daha sonra tekrar deneyin.",
  DUPLICATE: "Bu e-posta ile zaten bir hesap var.",
};

export function authErrorMessage(err: unknown, fallback = "İşlem başarısız"): string {
  const code = graphqlErrorCode(err);
  if (code && AUTH_ERROR_MESSAGES[code]) {
    return AUTH_ERROR_MESSAGES[code];
  }
  return fallback;
}

type MiyunaDesktop = {
  platform: string;
  getDeviceFingerprint: () => Promise<string>;
  getAppVersion: () => string;
  setRefreshToken?: (token: string) => Promise<boolean>;
  getRefreshToken?: () => Promise<string | null>;
  clearTokens?: () => Promise<boolean>;
  onDeepLink?: (callback: (url: string) => void) => void;
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

export function appVersion(): string | undefined {
  return window.miyunaDesktop?.getAppVersion?.();
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
  if (payload.errors?.length) {
    const code = payload.errors[0].extensions?.code ?? payload.errors[0].message;
    throw new GraphQLRequestError(code);
  }
  if (!payload.data) {
    throw new Error("GraphQL yanıtı boş");
  }
  return payload.data;
}

import { authErrorMessage as authErrorMessageForLocale } from "./i18n/auth-errors";
import { getStoredLocale, type Locale } from "./i18n/locale";
import { t } from "./i18n/messages";
import { resolveApiBaseUrl } from "./api-base";

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

function normalizeGraphqlErrorCode(code: string): string {
  return code.trim().toUpperCase();
}

const COMPLIANCE_ERROR_CODES = new Set([
  "CONSENT_REQUIRED",
  "COMPLIANCE_REJECTED",
  "COMPLIANCE_VIOLATION",
  "COMPLIANCE_BLOCKED",
]);

export function isAuthError(err: unknown): boolean {
  const code = graphqlErrorCode(err);
  if (!code) {
    return false;
  }
  const normalized = normalizeGraphqlErrorCode(code);
  return normalized === "UNAUTHORIZED" || normalized === "FORBIDDEN";
}

export function isComplianceErrorCode(code?: string | null): boolean {
  if (!code) return false;
  return COMPLIANCE_ERROR_CODES.has(normalizeGraphqlErrorCode(code));
}

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

export function graphqlErrorMessage(
  err: unknown,
  fallback?: string,
  locale: Locale = getStoredLocale(),
): string {
  const resolvedFallback = fallback ?? t("auth.fallback", locale);
  const code = graphqlErrorCode(err);
  if (code) {
    const mapped = authErrorMessageForLocale(normalizeGraphqlErrorCode(code), locale);
    if (mapped) {
      return mapped;
    }
  }
  if (isNetworkFailure(err)) {
    return t("auth.network", locale);
  }
  if (code) {
    return `${resolvedFallback} (${code})`;
  }
  return resolvedFallback;
}

export function authErrorMessage(err: unknown, fallback?: string, locale?: Locale): string {
  return graphqlErrorMessage(err, fallback, locale ?? getStoredLocale());
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

function graphqlUrl(): string {
  if (typeof window !== "undefined" && !window.miyunaDesktop) {
    return "/api/graphql";
  }
  return `${API_URL}/graphql`;
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
  const res = await fetch(graphqlUrl(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ query, variables }),
  });

  if (!res.ok) {
    const payload = (await res.json().catch(() => null)) as GraphQLResponse<T> | null;
    if (payload?.errors?.length) {
      const code = payload.errors[0].extensions?.code ?? payload.errors[0].message;
      throw new GraphQLRequestError(code);
    }
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

export function resolveApiBaseUrl(): string {
  const configured = process.env.NEXT_PUBLIC_API_URL?.trim();
  const raw = configured && configured.length > 0 ? configured : "http://localhost:8080";
  return raw.replace(/\/+$/, "").replace(/\/graphql$/i, "");
}

export function upstreamGraphqlUrl(): string {
  return `${resolveApiBaseUrl()}/graphql`;
}

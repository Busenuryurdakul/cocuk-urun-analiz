const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type GraphQLResponse<T> = {
  data?: T;
  errors?: Array<{ message: string; extensions?: { code?: string } }>;
};

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
    const code = payload.errors[0].extensions?.code;
    throw new Error(code ?? payload.errors[0].message);
  }
  if (!payload.data) {
    throw new Error("GraphQL yanıtı boş");
  }
  return payload.data;
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

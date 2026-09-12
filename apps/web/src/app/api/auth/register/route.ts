import { NextRequest, NextResponse } from "next/server";
import { mailhogMatchesRecipient } from "@/lib/mail-delivery-routing";

const VERIFY_RE = /https?:\/\/[^\s"'<>]+\/auth\/verify-email\?token=([A-Za-z0-9_-]+)/;

function apiBase() {
  return (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/+$/, "").replace(/\/graphql$/i, "");
}

async function localVerifyToken(email: string): Promise<string | null> {
  if (process.env.NODE_ENV === "production") return null;
  for (let attempt = 0; attempt < 6; attempt += 1) {
    try {
      const res = await fetch("http://127.0.0.1:8025/api/v2/messages?limit=20", { cache: "no-store" });
      if (res.ok) {
        const payload = (await res.json()) as {
          items?: Array<{ Content?: { Body?: string; Headers?: Record<string, string[]> }; Raw?: { Data?: string } }>;
        };
        for (const item of payload.items ?? []) {
          const to = (item.Content?.Headers?.To ?? []).join(" ").toLowerCase();
          const body = `${item.Content?.Body ?? ""}\n${item.Raw?.Data ?? ""}`;
          if (to && !mailhogMatchesRecipient(to, email)) continue;
          const match = body.match(VERIFY_RE);
          if (match?.[1]) return match[1];
        }
      }
    } catch {
      /* MailHog may still be catching up */
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  return null;
}

export async function POST(req: NextRequest) {
  const form = await req.formData();
  const email = String(form.get("email") ?? "").trim();
  const password = String(form.get("password") ?? "");
  const origin = req.nextUrl.origin;
  const fail = (reason: string) =>
    NextResponse.redirect(new URL(`/auth/register?error=${encodeURIComponent(reason)}`, origin), 303);

  if (!email || password.length < 8) {
    return fail("Geçerli bir e-posta ve en az 8 karakterlik şifre girin.");
  }

  try {
    const res = await fetch(`${apiBase()}/graphql`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: `mutation Register($input: RegisterInput!) { register(input: $input) { message } }`,
        variables: { input: { email, password } },
      }),
    });
    const payload = (await res.json()) as {
      data?: { register?: { message?: string } };
      errors?: Array<{ message?: string; extensions?: { code?: string } }>;
    };
    if (!res.ok || payload.errors?.length || !payload.data?.register) {
      const code = payload.errors?.[0]?.extensions?.code ?? payload.errors?.[0]?.message ?? "Kayıt başarısız";
      return fail(code === "INVALID_CREDENTIALS" ? "E-posta veya şifre hatalı." : "Kayıt başarısız");
    }
    const next = new URL("/auth/register", origin);
    next.searchParams.set("registered", "1");
    next.searchParams.set("email", email);
    if (process.env.NODE_ENV !== "production") {
      next.searchParams.set("local", "1");
      const token = await localVerifyToken(email);
      if (token) next.searchParams.set("token", token);
    }
    return NextResponse.redirect(next, 303);
  } catch {
    return fail("Sunucuya bağlanılamadı. Lütfen biraz sonra tekrar deneyin.");
  }
}

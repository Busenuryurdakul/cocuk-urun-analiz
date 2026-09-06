"use client";

import Link from "next/link";
import { FormEvent, Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AuthShell } from "@/components/layout/auth-shell";
import { TurnstileWidget, turnstileEnabled } from "@/components/turnstile-widget";
import { authErrorMessage, graphqlRequest } from "@/lib/graphql";

function isLocalHost() {
  if (typeof window === "undefined") return false;
  return window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1";
}

async function waitForLocalVerifyLink(email: string): Promise<string | null> {
  if (!isLocalHost()) return null;
  for (let attempt = 0; attempt < 8; attempt += 1) {
    try {
      const res = await fetch(`/api/dev/latest-verify-link?email=${encodeURIComponent(email)}`, {
        cache: "no-store",
      });
      if (res.ok) {
        const data = (await res.json()) as { url?: string | null };
        if (data.url) return data.url;
      }
    } catch {
      /* MailHog may still be receiving the message */
    }
    await new Promise((resolve) => setTimeout(resolve, 400));
  }
  return null;
}

function RegisterForm() {
  const params = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(params.get("registered") === "1");
  const [message, setMessage] = useState(
    params.get("registered") === "1" ? "Kayıt alındı. E-posta adresinize doğrulama bağlantısı gönderildi." : "",
  );
  const [error, setError] = useState(params.get("error") ?? "");
  const [email, setEmail] = useState(params.get("email") ?? "");
  const [password, setPassword] = useState("");
  const [turnstileToken, setTurnstileToken] = useState("");
  const [verifyLink, setVerifyLink] = useState<string | null>(
    params.get("token") ? `/auth/verify-email?token=${params.get("token")}` : null,
  );
  const localDev = params.get("local") === "1" || isLocalHost();

  useEffect(() => {
    if (!success || !email || verifyLink) return;
    void waitForLocalVerifyLink(email).then((link) => {
      if (link) setVerifyLink(link);
    });
  }, [success, email, verifyLink]);

  async function register() {
    if (email.trim() === "" || password.length < 8) {
      setError("Geçerli bir e-posta ve en az 8 karakterlik şifre girin.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await graphqlRequest<{ register: { message: string } }>(
        `mutation Register($input: RegisterInput!) {
          register(input: $input) { message }
        }`,
        {
          input: {
            email: email.trim(),
            password,
            turnstileToken: turnstileToken || null,
          },
        },
      );
      setMessage(data.register.message);
      setSuccess(true);
      setVerifyLink(await waitForLocalVerifyLink(email.trim()));
    } catch (err) {
      setError(authErrorMessage(err, "Kayıt başarısız"));
    } finally {
      setLoading(false);
    }
  }

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    await register();
  }

  return (
    <AuthShell title="Hesap oluştur" subtitle="E-posta doğrulama ve MFA ile korunan bir workspace açın.">
      {error && (
        <div role="alert" className="alert-error">
          {error}
        </div>
      )}

      {success ? (
        <div className="alert-success space-y-3">
          <p>{message}</p>
          <p>E-postanızdaki doğrulama bağlantısına tıklayın. Bağlantı 24 saat geçerlidir.</p>
          {localDev && (
            <p>
              Yerel geliştirmede e-posta{" "}
              <a className="font-semibold underline underline-offset-4" href="http://localhost:8025" target="_blank" rel="noreferrer">
                MailHog
              </a>{" "}
              kutusuna düşer; gerçek gelen kutunuza gitmez.
            </p>
          )}
          {verifyLink && (
            <Link className="inline-block font-semibold text-forest underline underline-offset-4" href={verifyLink}>
              E-postayı şimdi doğrula
            </Link>
          )}
          <Link className="block font-semibold text-forest underline underline-offset-4" href="/auth/login">
            Doğruladıktan sonra giriş yap
          </Link>
        </div>
      ) : (
        <form method="post" action="/api/auth/register" onSubmit={onSubmit} className="card space-y-4">
          <label className="label">
            E-posta
            <input
              name="email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="input"
            />
          </label>
          <label className="label">
            Şifre (min. 8 karakter)
            <input
              name="password"
              type="password"
              minLength={8}
              required
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input"
            />
          </label>
          {turnstileEnabled() && (
            <TurnstileWidget onToken={setTurnstileToken} onExpire={() => setTurnstileToken("")} />
          )}
          <button type="submit" disabled={loading} className="btn-primary w-full">
            {loading ? "Kaydediliyor…" : "Kayıt ol"}
          </button>
        </form>
      )}

      <p className="text-sm text-muted">
        Zaten hesabın var mı?{" "}
        <Link href="/auth/login" className="font-semibold text-forest underline underline-offset-4">
          Giriş yap
        </Link>
      </p>
    </AuthShell>
  );
}

export default function RegisterPage() {
  return (
    <Suspense fallback={<AuthShell title="Hesap oluştur"><p className="text-sm text-muted">Yükleniyor…</p></AuthShell>}>
      <RegisterForm />
    </Suspense>
  );
}

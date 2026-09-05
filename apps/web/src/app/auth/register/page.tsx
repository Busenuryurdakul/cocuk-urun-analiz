"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AuthShell } from "@/components/layout/auth-shell";
import { TurnstileWidget, turnstileEnabled } from "@/components/turnstile-widget";
import { authErrorMessage, graphqlRequest } from "@/lib/graphql";

export default function RegisterPage() {
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [turnstileToken, setTurnstileToken] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");
    const form = new FormData(e.currentTarget);
    try {
      const data = await graphqlRequest<{ register: { message: string } }>(
        `mutation Register($input: RegisterInput!) {
          register(input: $input) { message }
        }`,
        {
          input: {
            email: String(form.get("email")),
            password: String(form.get("password")),
            turnstileToken: turnstileToken || null,
          },
        },
      );
      setMessage(data.register.message);
      setSuccess(true);
    } catch (err) {
      setError(authErrorMessage(err, "Kayıt başarısız"));
    } finally {
      setLoading(false);
    }
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
          <Link className="inline-block font-semibold text-forest underline underline-offset-4" href="/auth/login">
            Doğruladıktan sonra giriş yap
          </Link>
        </div>
      ) : (
        <form onSubmit={onSubmit} className="card space-y-4">
          <label className="label">
            E-posta
            <input name="email" type="email" required className="input" />
          </label>
          <label className="label">
            Şifre (min. 8 karakter)
            <input name="password" type="password" minLength={8} required className="input" />
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

"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { TurnstileWidget, turnstileEnabled } from "@/components/turnstile-widget";
import { authErrorMessage, clientPlatform, deviceFingerprintAsync, graphqlRequest } from "@/lib/graphql";

type LoginStatus =
  | "EMAIL_OTP_REQUIRED"
  | "MFA_REQUIRED"
  | "AUTHENTICATED";

export default function EmailOTPPage() {
  const router = useRouter();
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");
  const [resending, setResending] = useState(false);
  const [error, setError] = useState("");
  const [turnstileToken, setTurnstileToken] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    const code = new FormData(e.currentTarget).get("code") as string;
    const fingerprint = await deviceFingerprintAsync();
    try {
      const data = await graphqlRequest<{ verifyLoginEmailOTP: { status: LoginStatus } }>(
        `mutation VerifyLoginEmailOTP($input: VerifyLoginEmailOTPInput!) {
          verifyLoginEmailOTP(input: $input) { status }
        }`,
        {
          input: {
            code,
            deviceFingerprint: fingerprint,
            turnstileToken: turnstileToken || null,
          },
        },
      );
      switch (data.verifyLoginEmailOTP.status) {
        case "MFA_REQUIRED":
          router.push("/auth/mfa?step=verify");
          break;
        default:
          router.push("/workspace");
      }
    } catch (err) {
      setError(authErrorMessage(err, "Doğrulama kodu geçersiz"));
      setState("error");
    }
  }

  async function onResend() {
    setResending(true);
    setError("");
    try {
      await graphqlRequest(`mutation { resendLoginEmailOTP { status } }`);
      setState("idle");
    } catch (err) {
      setError(authErrorMessage(err, "Kod gönderilemedi"));
    } finally {
      setResending(false);
    }
  }

  return (
    <AuthShell title="E-posta doğrulama" subtitle="Giriş için e-postanıza gönderilen 6 haneli kodu girin.">
      {state === "loading" && <AsyncView state="loading" />}
      {(state === "idle" || state === "error") && (
        <form onSubmit={onSubmit} className="card space-y-4">
          {error && <p className="alert-error">{error}</p>}
          <label className="label">
            Doğrulama kodu
            <input name="code" required className="input" autoComplete="one-time-code" inputMode="numeric" pattern="[0-9]{6}" maxLength={6} />
          </label>
          {turnstileEnabled() && (
            <TurnstileWidget onToken={setTurnstileToken} onExpire={() => setTurnstileToken("")} />
          )}
          <button type="submit" className="btn-primary w-full">
            Doğrula
          </button>
          <button type="button" onClick={() => void onResend()} disabled={resending} className="btn-secondary w-full">
            {resending ? "Gönderiliyor…" : "Kodu tekrar gönder"}
          </button>
        </form>
      )}
      <Link href="/auth/login" className="link-quiet">
        Giriş sayfası
      </Link>
    </AuthShell>
  );
}

"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { TurnstileWidget, turnstileEnabled } from "@/components/turnstile-widget";
import { authErrorMessage, deviceFingerprintAsync, graphqlRequest } from "@/lib/graphql";
import { clearPendingToken, readPendingToken, savePendingToken } from "@/lib/pending-auth";

type LoginStatus =
  | "EMAIL_OTP_REQUIRED"
  | "MFA_REQUIRED"
  | "AUTHENTICATED";

export default function EmailOTPPage() {
  const router = useRouter();
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");
  const [resending, setResending] = useState(false);
  const [error, setError] = useState("");
  const [info, setInfo] = useState("");
  const [turnstileToken, setTurnstileToken] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    setInfo("");
    const code = new FormData(e.currentTarget).get("code") as string;
    const fingerprint = await deviceFingerprintAsync();
    const pendingToken = readPendingToken();
    try {
      const data = await graphqlRequest<{ verifyLoginEmailOTP: { status: LoginStatus; pendingToken?: string | null } }>(
        `mutation VerifyLoginEmailOTP($input: VerifyLoginEmailOTPInput!) {
          verifyLoginEmailOTP(input: $input) { status pendingToken }
        }`,
        {
          input: {
            code,
            deviceFingerprint: fingerprint,
            turnstileToken: turnstileToken || null,
            pendingToken,
          },
        },
      );
      savePendingToken(data.verifyLoginEmailOTP.pendingToken);
      switch (data.verifyLoginEmailOTP.status) {
        case "MFA_REQUIRED":
          router.push("/auth/mfa?step=verify");
          break;
        default:
          clearPendingToken();
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
    setInfo("");
    const pendingToken = readPendingToken();
    try {
      const data = await graphqlRequest<{ resendLoginEmailOTP: { status: LoginStatus; pendingToken?: string | null } }>(
        `mutation ResendLoginEmailOTP($input: ResendLoginEmailOTPInput!) {
          resendLoginEmailOTP(input: $input) { status pendingToken }
        }`,
        { input: { pendingToken } },
      );
      savePendingToken(data.resendLoginEmailOTP.pendingToken);
      setState("idle");
      setInfo("Yeni doğrulama kodu e-posta adresinize gönderildi. Gelen kutusu ve spam klasörünü kontrol edin.");
    } catch (err) {
      setError(authErrorMessage(err, "Kod gönderilemedi. Giriş sayfasından tekrar deneyin."));
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
          {info && <p className="alert-success">{info}</p>}
          {!info && !error && (
            <p className="text-sm text-muted">
              Kod birkaç dakika içinde gelmezse spam klasörünü kontrol edin veya aşağıdan tekrar gönderin.
            </p>
          )}
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

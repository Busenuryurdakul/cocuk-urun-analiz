"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AuthShell } from "@/components/layout/auth-shell";
import { TurnstileWidget, turnstileEnabled } from "@/components/turnstile-widget";
import {
  appVersion,
  authErrorMessage,
  clientPlatform,
  deviceFingerprintAsync,
  graphqlErrorCode,
  graphqlRequest,
} from "@/lib/graphql";

type LoginStatus =
  | "EMAIL_OTP_REQUIRED"
  | "MFA_SETUP_REQUIRED"
  | "MFA_REQUIRED"
  | "DEVICE_VERIFICATION_REQUIRED"
  | "AUTHENTICATED";

export default function LoginPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [resending, setResending] = useState(false);
  const [error, setError] = useState("");
  const [info, setInfo] = useState("");
  const [pendingVerify, setPendingVerify] = useState<{ email: string; password: string } | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [turnstileToken, setTurnstileToken] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    setError("");
    setInfo("");
    setPendingVerify(null);
    const fingerprint = await deviceFingerprintAsync();
    const version = await appVersion();
    try {
      const data = await graphqlRequest<{ login: { status: LoginStatus; mfaSetup?: { secret: string; otpauthUrl: string } } }>(
        `mutation Login($input: LoginInput!) {
          login(input: $input) { status mfaSetup { secret otpauthUrl } }
        }`,
        {
          input: {
            email,
            password,
            deviceFingerprint: fingerprint,
            platform: clientPlatform(),
            appVersion: version ?? null,
            turnstileToken: turnstileToken || null,
          },
        },
      );
      switch (data.login.status) {
        case "EMAIL_OTP_REQUIRED":
          router.push("/auth/email-otp");
          break;
        case "MFA_SETUP_REQUIRED":
          if (data.login.mfaSetup) {
            sessionStorage.setItem("miyuna_mfa_setup", JSON.stringify(data.login.mfaSetup));
          }
          router.push("/auth/mfa");
          break;
        case "MFA_REQUIRED":
          router.push("/auth/mfa?step=verify");
          break;
        case "DEVICE_VERIFICATION_REQUIRED":
          router.push("/auth/email-otp");
          break;
        default:
          router.push("/workspace");
      }
    } catch (err) {
      if (graphqlErrorCode(err) === "EMAIL_NOT_VERIFIED") {
        setPendingVerify({ email, password });
      }
      setError(authErrorMessage(err, "Giriş başarısız"));
    } finally {
      setLoading(false);
    }
  }

  async function onResend() {
    setResending(true);
    setInfo("");
    const creds = pendingVerify ?? { email, password };
    try {
      const data = await graphqlRequest<{ resendEmailVerification: { message: string } }>(
        `mutation ResendEmailVerification($input: ResendEmailVerificationInput!) {
          resendEmailVerification(input: $input) { message }
        }`,
        {
          input: {
            email: creds.email,
            password: creds.password,
            turnstileToken: turnstileToken || null,
          },
        },
      );
      setError("");
      setInfo(data.resendEmailVerification.message);
    } catch (err) {
      setError(authErrorMessage(err, "Doğrulama e-postası gönderilemedi"));
    } finally {
      setResending(false);
    }
  }

  return (
    <AuthShell title="Giriş yap" subtitle="Çalışma alanınıza ve kanıtlı ürün analizine dönün.">
      {error && (
        <div role="alert" className="alert-error">
          <p>{error}</p>
          {pendingVerify && (
            <button
              type="button"
              onClick={() => void onResend()}
              disabled={resending}
              className="mt-3 text-sm font-semibold text-forest underline disabled:opacity-50"
            >
              {resending ? "Gönderiliyor…" : "Doğrulama e-postasını tekrar gönder"}
            </button>
          )}
        </div>
      )}

      {info && (
        <div role="status" className="alert-success">
          {info}
        </div>
      )}

      <form method="post" onSubmit={onSubmit} className="card space-y-4">
        <label className="label">
          E-posta
          <input
            name="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="input"
          />
        </label>
        <label className="label">
          Şifre
          <input
            name="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="input"
          />
        </label>
        {turnstileEnabled() && (
          <TurnstileWidget onToken={setTurnstileToken} onExpire={() => setTurnstileToken("")} />
        )}
        <button type="submit" disabled={loading} className="btn-primary w-full">
          {loading ? "Giriş yapılıyor…" : "Giriş yap"}
        </button>
      </form>

      <p className="text-sm text-muted">
        Hesabın yok mu?{" "}
        <Link href="/auth/register" className="font-semibold text-forest underline underline-offset-4">
          Kayıt ol
        </Link>
      </p>
    </AuthShell>
  );
}

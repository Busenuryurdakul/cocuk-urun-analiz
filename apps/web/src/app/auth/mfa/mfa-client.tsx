"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { authErrorMessage, deviceFingerprintAsync, graphqlRequest } from "@/lib/graphql";

export default function MFAPageClient() {
  const router = useRouter();
  const params = useSearchParams();
  const step = params.get("step");
  const isVerify = step === "verify";
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    const code = new FormData(e.currentTarget).get("code") as string;
    const fingerprint = await deviceFingerprintAsync();
    try {
      if (isVerify) {
        const data = await graphqlRequest<{ verifyLoginMFA: { status: string } }>(
          `mutation VerifyLoginMFA($input: VerifyLoginMFAInput!) {
            verifyLoginMFA(input: $input) { status }
          }`,
          { input: { code, deviceFingerprint: fingerprint } },
        );
        if (data.verifyLoginMFA.status === "EMAIL_OTP_REQUIRED") {
          router.push("/auth/email-otp");
        } else {
          router.push("/workspace");
        }
      } else {
        await graphqlRequest(`mutation ConfirmMFA($input: ConfirmMFAInput!) {
          confirmMFA(input: $input)
        }`, { input: { code } });
        router.push("/auth/login");
      }
    } catch (err) {
      setError(authErrorMessage(err, "MFA doğrulama başarısız"));
      setState("error");
    }
  }

  return (
    <AuthShell
      title={isVerify ? "MFA doğrulama" : "MFA kurulumu"}
      subtitle="Authenticator uygulamanızdaki 6 haneli kodu girin."
    >
      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && <AsyncView state="error" error={<p className="alert-error">{error}</p>} />}
      {state === "idle" && (
        <form onSubmit={onSubmit} className="card space-y-4">
          <label className="label">
            Authenticator kodu
            <input name="code" required className="input" autoComplete="one-time-code" />
          </label>
          <button type="submit" className="btn-primary w-full">
            Doğrula
          </button>
        </form>
      )}
      <Link href="/auth/login" className="link-quiet">
        Giriş sayfası
      </Link>
    </AuthShell>
  );
}

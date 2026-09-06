"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { MfaSetupPanel } from "@/components/mfa-setup-panel";
import { authErrorMessage, deviceFingerprintAsync, graphqlRequest } from "@/lib/graphql";

type MfaSetupInfo = {
  secret: string;
  otpauthUrl: string;
  setupToken: string;
};

export default function MFAPageClient() {
  const router = useRouter();
  const params = useSearchParams();
  const step = params.get("step");
  const isVerify = step === "verify";
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");
  const [setupState, setSetupState] = useState<"loading" | "ready" | "error">("loading");
  const [setup, setSetup] = useState<MfaSetupInfo | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (isVerify) {
      setSetupState("ready");
      return;
    }
    const cached = sessionStorage.getItem("miyuna_mfa_setup");
    if (cached) {
      try {
        setSetup(JSON.parse(cached) as MfaSetupInfo);
        setSetupState("ready");
        return;
      } catch {
        sessionStorage.removeItem("miyuna_mfa_setup");
      }
    }
    graphqlRequest<{ pendingMfaSetup: MfaSetupInfo }>(
      `query PendingMfaSetup {
        pendingMfaSetup { secret otpauthUrl setupToken }
      }`,
    )
      .then((data) => {
        setSetup(data.pendingMfaSetup);
        setSetupState("ready");
      })
      .catch(() => setSetupState("error"));
  }, [isVerify]);

  async function confirmSetup(code: string) {
    setState("loading");
    try {
      const setupToken = setup?.setupToken ?? null;
      await graphqlRequest(`mutation ConfirmMFA($input: ConfirmMFAInput!) {
        confirmMFA(input: $input)
      }`, { input: { code, setupToken } });
      sessionStorage.removeItem("miyuna_mfa_setup");
      router.push("/auth/login");
    } catch (err) {
      setError(authErrorMessage(err, "MFA doğrulama başarısız"));
      setState("error");
    }
  }

  async function onVerifySubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    const code = new FormData(e.currentTarget).get("code") as string;
    const fingerprint = await deviceFingerprintAsync();
    try {
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
    } catch (err) {
      setError(authErrorMessage(err, "MFA doğrulama başarısız"));
      setState("error");
    }
  }

  return (
    <AuthShell
      title={isVerify ? "MFA doğrulama" : "MFA kurulumu"}
      subtitle={
        isVerify
          ? "Authenticator uygulamanızdaki 6 haneli kodu girin."
          : "Authenticator uygulamanızla QR kodu okutun, ardından kodu girin."
      }
    >
      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && <AsyncView state="error" error={<p className="alert-error">{error}</p>} />}
      {state === "idle" && isVerify && (
        <form onSubmit={onVerifySubmit} className="card space-y-4">
          <label className="label">
            Authenticator kodu
            <input name="code" required className="input" autoComplete="one-time-code" inputMode="numeric" />
          </label>
          <button type="submit" className="btn-primary w-full">
            Doğrula
          </button>
        </form>
      )}
      {state === "idle" && !isVerify && setupState === "loading" && <AsyncView state="loading" />}
      {state === "idle" && !isVerify && setupState === "error" && (
        <AsyncView
          state="error"
          error={
            <p className="alert-error">
              MFA kurulum oturumu bulunamadı. Lütfen çıkış yapıp tekrar giriş deneyin.
            </p>
          }
        />
      )}
      {state === "idle" && !isVerify && setupState === "ready" && setup && (
        <MfaSetupPanel secret={setup.secret} otpauthUrl={setup.otpauthUrl} onSubmit={confirmSetup} />
      )}
      <Link href="/auth/login" className="link-quiet">
        Giriş sayfası
      </Link>
    </AuthShell>
  );
}

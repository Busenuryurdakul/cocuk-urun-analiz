"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { graphqlRequest } from "@/lib/graphql";

export default function VerifyEmailClient() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token");
  const [state, setState] = useState<"loading" | "success" | "error" | "empty">("loading");
  const [secret, setSecret] = useState("");

  useEffect(() => {
    if (!token) {
      setState("empty");
      return;
    }
    graphqlRequest<{ verifyEmail: { secret: string } }>(
      `mutation VerifyEmail($token: String!) {
        verifyEmail(token: $token) { secret }
      }`,
      { token },
    )
      .then((data) => {
        setSecret(data.verifyEmail.secret);
        setState("success");
      })
      .catch(() => setState("error"));
  }, [token]);

  async function onConfirm(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    try {
      await graphqlRequest(`mutation ConfirmMFA($input: ConfirmMFAInput!) {
        confirmMFA(input: $input)
      }`, { input: { code: String(form.get("code")) } });
      router.push("/auth/login");
    } catch {
      setState("error");
    }
  }

  return (
    <AuthShell title="E-posta doğrulama" subtitle="Hesabınızı doğrulayın ve MFA’yı etkinleştirin.">
      {state === "loading" && <AsyncView state="loading" />}
      {state === "empty" && <AsyncView state="empty" empty={<p className="card text-sm text-muted">Doğrulama bağlantısı eksik.</p>} />}
      {state === "error" && <AsyncView state="error" />}
      {state === "success" && (
        <div className="card space-y-4">
          <p className="text-sm text-muted">
            MFA kurulumu için authenticator uygulamanıza aşağıdaki secret&apos;ı ekleyin:
          </p>
          <code className="block break-all rounded-xl bg-cream p-3 text-xs">{secret}</code>
          <form onSubmit={onConfirm} className="space-y-3">
            <label className="label">
              Authenticator kodu
              <input name="code" required className="input" autoComplete="one-time-code" />
            </label>
            <button type="submit" className="btn-primary w-full">
              MFA&apos;yı etkinleştir
            </button>
          </form>
        </div>
      )}
      <Link href="/auth/login" className="link-quiet">
        Giriş sayfası
      </Link>
    </AuthShell>
  );
}

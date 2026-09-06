"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { MfaSetupPanel } from "@/components/mfa-setup-panel";
import { graphqlRequest } from "@/lib/graphql";

export default function VerifyEmailClient() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token");
  const [state, setState] = useState<"loading" | "success" | "error" | "empty">("loading");
  const [secret, setSecret] = useState("");
  const [otpauthUrl, setOtpauthUrl] = useState("");

  useEffect(() => {
    if (!token) {
      setState("empty");
      return;
    }
    graphqlRequest<{ verifyEmail: { secret: string; otpauthUrl: string } }>(
      `mutation VerifyEmail($token: String!) {
        verifyEmail(token: $token) { secret otpauthUrl }
      }`,
      { token },
    )
      .then((data) => {
        setSecret(data.verifyEmail.secret);
        setOtpauthUrl(data.verifyEmail.otpauthUrl);
        setState("success");
      })
      .catch(() => setState("error"));
  }, [token]);

  async function onConfirm(code: string) {
    try {
      await graphqlRequest(`mutation ConfirmMFA($input: ConfirmMFAInput!) {
        confirmMFA(input: $input)
      }`, { input: { code } });
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
        <MfaSetupPanel
          secret={secret}
          otpauthUrl={otpauthUrl}
          submitLabel="MFA'yı etkinleştir"
          onSubmit={onConfirm}
        />
      )}
      <Link href="/auth/login" className="link-quiet">
        Giriş sayfası
      </Link>
    </AuthShell>
  );
}

"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
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
    <main className="mx-auto flex min-h-screen max-w-md flex-col gap-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">E-posta doğrulama</h1>
      {state === "loading" && <AsyncView state="loading" />}
      {state === "empty" && <AsyncView state="empty" empty={<p>Doğrulama bağlantısı eksik.</p>} />}
      {state === "error" && <AsyncView state="error" />}
      {state === "success" && (
        <div className="space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <p className="text-sm text-slate-600">
            MFA kurulumu için authenticator uygulamanıza aşağıdaki secret&apos;ı ekleyin:
          </p>
          <code className="block break-all rounded bg-slate-100 p-3 text-xs">{secret}</code>
          <form onSubmit={onConfirm} className="space-y-3">
            <label className="block text-sm">
              Authenticator kodu
              <input
                name="code"
                required
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              />
            </label>
            <button type="submit" className="rounded-md bg-miyuna-600 px-4 py-2 text-sm text-white">
              MFA&apos;yı etkinleştir
            </button>
          </form>
        </div>
      )}
      <Link href="/auth/login" className="text-sm underline">
        Giriş sayfası
      </Link>
    </main>
  );
}

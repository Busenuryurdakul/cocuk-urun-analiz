"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { deviceFingerprint, graphqlRequest } from "@/lib/graphql";

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
    const fingerprint = deviceFingerprint();
    try {
      if (isVerify) {
        const data = await graphqlRequest<{ verifyLoginMFA: { status: string } }>(
          `mutation VerifyLoginMFA($input: VerifyLoginMFAInput!) {
            verifyLoginMFA(input: $input) { status }
          }`,
          { input: { code, deviceFingerprint: fingerprint } },
        );
        if (data.verifyLoginMFA.status === "DEVICE_VERIFICATION_REQUIRED") {
          router.push("/auth/device");
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
      setError(err instanceof Error ? err.message : "MFA doğrulama başarısız");
      setState("error");
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col gap-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">{isVerify ? "MFA doğrulama" : "MFA kurulumu"}</h1>
      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && (
        <AsyncView state="error" error={<p className="text-sm text-red-800">{error}</p>} />
      )}
      {state === "idle" && (
        <form onSubmit={onSubmit} className="space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <label className="block text-sm">
            Authenticator kodu
            <input name="code" required className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" />
          </label>
          <button type="submit" className="w-full rounded-md bg-miyuna-600 px-4 py-2 text-sm text-white">
            Doğrula
          </button>
        </form>
      )}
      <Link href="/auth/login" className="text-sm underline">
        Giriş sayfası
      </Link>
    </main>
  );
}

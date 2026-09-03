"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { deviceFingerprint, graphqlRequest } from "@/lib/graphql";

type LoginStatus =
  | "MFA_SETUP_REQUIRED"
  | "MFA_REQUIRED"
  | "DEVICE_VERIFICATION_REQUIRED"
  | "AUTHENTICATED";

export default function LoginPage() {
  const router = useRouter();
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    const form = new FormData(e.currentTarget);
    const fingerprint = deviceFingerprint();
    try {
      const data = await graphqlRequest<{ login: { status: LoginStatus } }>(
        `mutation Login($input: LoginInput!) {
          login(input: $input) { status }
        }`,
        {
          input: {
            email: String(form.get("email")),
            password: String(form.get("password")),
            deviceFingerprint: fingerprint,
          },
        },
      );
      switch (data.login.status) {
        case "MFA_SETUP_REQUIRED":
          router.push("/auth/mfa");
          break;
        case "MFA_REQUIRED":
          router.push("/auth/mfa?step=verify");
          break;
        case "DEVICE_VERIFICATION_REQUIRED":
          router.push("/auth/device");
          break;
        default:
          router.push("/workspace");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Giriş başarısız");
      setState("error");
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col gap-6 px-6 py-16">
      <header>
        <p className="text-sm font-medium uppercase tracking-wide text-miyuna-600">Miyuna</p>
        <h1 className="text-2xl font-semibold text-slate-900">Giriş yap</h1>
      </header>

      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && (
        <AsyncView state="error" error={<p className="text-sm text-red-800">{error}</p>} />
      )}
      {state === "idle" && (
        <form onSubmit={onSubmit} className="space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <label className="block text-sm">
            E-posta
            <input
              name="email"
              type="email"
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            />
          </label>
          <label className="block text-sm">
            Şifre
            <input
              name="password"
              type="password"
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            />
          </label>
          <button
            type="submit"
            className="w-full rounded-md bg-miyuna-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700"
          >
            Giriş yap
          </button>
        </form>
      )}

      <Link href="/auth/register" className="text-sm text-slate-600 underline">
        Hesabın yok mu? Kayıt ol
      </Link>
    </main>
  );
}

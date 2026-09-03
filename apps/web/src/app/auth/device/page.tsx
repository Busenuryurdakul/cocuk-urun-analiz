"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { deviceFingerprint, graphqlRequest } from "@/lib/graphql";

export default function DeviceVerifyPage() {
  const router = useRouter();
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    const code = new FormData(e.currentTarget).get("code") as string;
    try {
      await graphqlRequest(`mutation VerifyDevice($input: VerifyDeviceInput!) {
        verifyDevice(input: $input) { status }
      }`, {
        input: { code, deviceFingerprint: deviceFingerprint() },
      });
      router.push("/workspace");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Cihaz doğrulama başarısız");
      setState("error");
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col gap-6 px-6 py-16">
      <h1 className="text-2xl font-semibold">Cihaz doğrulama</h1>
      <p className="text-sm text-slate-600">E-postanıza gönderilen 6 haneli kodu girin.</p>
      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && (
        <AsyncView state="error" error={<p className="text-sm text-red-800">{error}</p>} />
      )}
      {state === "idle" && (
        <form onSubmit={onSubmit} className="space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <label className="block text-sm">
            Doğrulama kodu
            <input name="code" required className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2" />
          </label>
          <button type="submit" className="w-full rounded-md bg-miyuna-600 px-4 py-2 text-sm text-white">
            Cihazı doğrula
          </button>
        </form>
      )}
      <Link href="/auth/login" className="text-sm underline">
        Giriş sayfası
      </Link>
    </main>
  );
}

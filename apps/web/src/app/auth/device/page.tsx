"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { authErrorMessage, deviceFingerprintAsync, graphqlRequest } from "@/lib/graphql";

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
        input: { code, deviceFingerprint: await deviceFingerprintAsync() },
      });
      router.push("/workspace");
    } catch (err) {
      setError(authErrorMessage(err, "Cihaz doğrulama başarısız"));
      setState("error");
    }
  }

  return (
    <AuthShell title="Cihaz doğrulama" subtitle="E-postanıza gönderilen 6 haneli kodu girin.">
      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && <AsyncView state="error" error={<p className="alert-error">{error}</p>} />}
      {state === "idle" && (
        <form onSubmit={onSubmit} className="card space-y-4">
          <label className="label">
            Doğrulama kodu
            <input name="code" required className="input" autoComplete="one-time-code" />
          </label>
          <button type="submit" className="btn-primary w-full">
            Cihazı doğrula
          </button>
        </form>
      )}
      <Link href="/auth/login" className="link-quiet">
        Giriş sayfası
      </Link>
    </AuthShell>
  );
}

import { Suspense } from "react";
import VerifyEmailClient from "./verify-email-client";

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<p className="p-16 text-sm text-slate-600">Yükleniyor…</p>}>
      <VerifyEmailClient />
    </Suspense>
  );
}

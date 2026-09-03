import { Suspense } from "react";
import MFAPageClient from "./mfa-client";

export default function MFAPage() {
  return (
    <Suspense fallback={<p className="p-16 text-sm text-slate-600">Yükleniyor…</p>}>
      <MFAPageClient />
    </Suspense>
  );
}

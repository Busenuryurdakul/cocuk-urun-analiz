import { Suspense } from "react";
import MFAPageClient from "./mfa-client";

export default function MFAPage() {
  return (
    <Suspense fallback={<p className="p-16 text-sm text-muted">Yükleniyor…</p>}>
      <MFAPageClient />
    </Suspense>
  );
}

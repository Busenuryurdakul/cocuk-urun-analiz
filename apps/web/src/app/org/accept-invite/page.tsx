import { Suspense } from "react";
import AcceptInviteClient from "./accept-invite-client";

export default function AcceptInvitePage() {
  return (
    <Suspense fallback={<p className="p-16 text-sm text-muted">Yükleniyor…</p>}>
      <AcceptInviteClient />
    </Suspense>
  );
}

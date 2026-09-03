import { Suspense } from "react";
import AcceptInviteClient from "./accept-invite-client";

export default function AcceptInvitePage() {
  return (
    <Suspense fallback={<main className="mx-auto px-6 py-16">Yükleniyor…</main>}>
      <AcceptInviteClient />
    </Suspense>
  );
}

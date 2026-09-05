"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AuthShell } from "@/components/layout/auth-shell";
import { graphqlRequest } from "@/lib/graphql";

export default function AcceptInviteClient() {
  const search = useSearchParams();
  const token = search.get("token") ?? "";
  const [state, setState] = useState<"loading" | "success" | "error" | "unauthorized">("loading");
  const [orgId, setOrgId] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      setState("error");
      return;
    }
    graphqlRequest<{ acceptInvitation: { id: string; name: string } }>(
      `mutation($token: String!) { acceptInvitation(token: $token) { id name } }`,
      { token },
    )
      .then((data) => {
        setOrgId(data.acceptInvitation.id);
        setState("success");
      })
      .catch((err) => {
        if (err instanceof Error && err.message === "UNAUTHORIZED") {
          setState("unauthorized");
        } else {
          setState("error");
        }
      });
  }, [token]);

  return (
    <AuthShell title="Davet kabul" subtitle="Organizasyon davetini onaylayın.">
      {state === "loading" && <AsyncView state="loading" />}
      {state === "unauthorized" && (
        <AsyncView
          state="unauthorized"
          unauthorized={
            <Link href={`/auth/login?next=/org/accept-invite?token=${encodeURIComponent(token)}`} className="btn-primary">
              Giriş yapın
            </Link>
          }
        />
      )}
      {state === "error" && <AsyncView state="error" />}
      {state === "success" && orgId && (
        <div className="alert-success">
          <p>Davet kabul edildi.</p>
          <Link href={`/org/${orgId}/members`} className="mt-3 inline-block font-semibold underline">
            Organizasyona git
          </Link>
        </div>
      )}
    </AuthShell>
  );
}

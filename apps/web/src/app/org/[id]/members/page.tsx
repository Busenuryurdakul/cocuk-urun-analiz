"use client";

import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { graphqlRequest } from "@/lib/graphql";

type Member = {
  userId: string;
  email: string;
  role: string;
};

export default function OrgMembersPage() {
  const params = useParams<{ id: string }>();
  const orgId = params.id;
  const [members, setMembers] = useState<Member[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("VIEWER");
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized" | "empty">("loading");

  const load = useCallback(async () => {
    setView("loading");
    try {
      const data = await graphqlRequest<{ organizationMembers: Member[] }>(
        `query($id: ID!) { organizationMembers(organizationId: $id) { userId email role } }`,
        { id: orgId },
      );
      setMembers(data.organizationMembers);
      setView(data.organizationMembers.length === 0 ? "empty" : "success");
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setView("unauthorized");
      } else {
        setView("error");
      }
    }
  }, [orgId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function invite(e: FormEvent) {
    e.preventDefault();
    await graphqlRequest(
      `mutation($input: InviteMemberInput!) { inviteMember(input: $input) }`,
      { input: { organizationId: orgId, email, role } },
    );
    setEmail("");
    await load();
  }

  return (
    <AppShell
      title="Üyeler"
      kicker="Organizasyon"
      orgId={orgId}
      description="Davet gönderin; roller OWNER, ADMIN, ANALYST veya VIEWER olabilir."
    >
      {view === "loading" && <AsyncView state="loading" />}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && (
        <AsyncView
          state="retry"
          retry={
            <button type="button" onClick={() => void load()} className="btn-secondary">
              Tekrar dene
            </button>
          }
        />
      )}
      {view === "empty" && <AsyncView state="empty" />}
      {members.length > 0 && (
        <ul className="card divide-y divide-sand text-sm">
          {members.map((m) => (
            <li key={m.userId} className="flex items-center justify-between py-3 first:pt-0 last:pb-0">
              <span>{m.email}</span>
              <span className="badge-muted">{m.role}</span>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={invite} className="card mt-6 max-w-lg space-y-3">
        <h2 className="font-display text-xl">Davet gönder</h2>
        <input className="input" placeholder="E-posta" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <select className="input" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="ADMIN">ADMIN</option>
          <option value="ANALYST">ANALYST</option>
          <option value="VIEWER">VIEWER</option>
        </select>
        <button type="submit" className="btn-primary">
          Davet et
        </button>
      </form>
    </AppShell>
  );
}

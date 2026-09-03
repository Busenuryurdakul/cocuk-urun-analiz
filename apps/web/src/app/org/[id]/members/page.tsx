"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
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
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col gap-6 px-6 py-16">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Üyeler</h1>
        <Link href={`/org/${orgId}/compliance`} className="text-sm text-miyuna-600 underline">
          Uyumluluk
        </Link>
      </header>
      {view === "loading" && <AsyncView state="loading" />}
      {view === "unauthorized" && <AsyncView state="unauthorized" />}
      {view === "error" && <AsyncView state="retry" retry={<button type="button" onClick={() => void load()} className="underline">Tekrar dene</button>} />}
      {view === "empty" && <AsyncView state="empty" />}
      {members.length > 0 && (
        <ul className="space-y-2 rounded-xl border bg-white p-4 text-sm">
          {members.map((m) => (
            <li key={m.userId} className="flex justify-between border-b border-slate-100 py-2 last:border-0">
              <span>{m.email}</span>
              <span className="text-slate-500">{m.role}</span>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={invite} className="space-y-3 rounded-xl border bg-white p-4">
        <h2 className="font-medium">Davet gönder</h2>
        <input className="w-full rounded border px-3 py-2 text-sm" placeholder="E-posta" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <select className="w-full rounded border px-3 py-2 text-sm" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="ADMIN">ADMIN</option>
          <option value="ANALYST">ANALYST</option>
          <option value="VIEWER">VIEWER</option>
        </select>
        <button type="submit" className="rounded-md bg-miyuna-600 px-4 py-2 text-sm text-white">Davet et</button>
      </form>
      <Link href="/workspace" className="text-sm text-slate-500 underline">Workspace</Link>
    </main>
  );
}

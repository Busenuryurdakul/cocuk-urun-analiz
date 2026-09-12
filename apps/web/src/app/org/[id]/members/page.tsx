"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { AppShell } from "@/components/layout/app-shell";
import { authErrorMessage, graphqlErrorCode, graphqlRequest } from "@/lib/graphql";

type Member = {
  userId: string;
  email: string;
  role: string;
};

type Organization = {
  id: string;
  name: string;
  type: string;
};

export default function OrgMembersPage() {
  const params = useParams<{ id: string }>();
  const orgId = params.id;
  const [org, setOrg] = useState<Organization | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("VIEWER");
  const [view, setView] = useState<"loading" | "success" | "error" | "unauthorized" | "empty">("loading");
  const [status, setStatus] = useState<{ kind: "idle" | "pending" | "success" | "error"; text: string }>({
    kind: "idle",
    text: "",
  });
  const [submitting, setSubmitting] = useState(false);

  const isPersonal = org?.type === "PERSONAL";

  const load = useCallback(async (opts?: { silent?: boolean }) => {
    if (!opts?.silent) {
      setView("loading");
    }
    try {
      const data = await graphqlRequest<{
        organization: Organization;
        organizationMembers: Member[];
      }>(
        `query($id: ID!) {
          organization(organizationId: $id) { id name type }
          organizationMembers(organizationId: $id) { userId email role }
        }`,
        { id: orgId },
      );
      setOrg(data.organization);
      setMembers(data.organizationMembers);
      setView(data.organizationMembers.length === 0 ? "empty" : "success");
    } catch (err) {
      if (err instanceof Error && (err.message === "UNAUTHORIZED" || err.message === "FORBIDDEN")) {
        setView("unauthorized");
      } else if (!opts?.silent) {
        setView("error");
      }
    }
  }, [orgId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function sendInvite(trimmed: string) {
    await graphqlRequest(
      `mutation($input: InviteMemberInput!) { inviteMember(input: $input) }`,
      { input: { organizationId: orgId, email: trimmed, role } },
    );
  }

  async function ensureInviteConsents() {
    for (const purpose of ["REGISTRATION", "DATA_PROCESSING", "ORG_MEMBERSHIP"]) {
      try {
        await graphqlRequest(
          `mutation($input: GrantConsentInput!) { grantConsent(input: $input) { id } }`,
          { input: { purpose, organizationId: orgId } },
        );
      } catch {
        // Already granted or not applicable for this profile.
      }
    }
  }

  async function invite(e: FormEvent) {
    e.preventDefault();
    const trimmed = email.trim();
    if (!trimmed || !trimmed.includes("@")) {
      setStatus({ kind: "error", text: "Geçerli bir e-posta girin." });
      return;
    }
    if (members.some((m) => m.email.toLowerCase() === trimmed.toLowerCase())) {
      setStatus({ kind: "error", text: "Bu e-posta zaten bu organizasyonun üyesi. Başka bir e-posta girin." });
      return;
    }

    setSubmitting(true);
    setStatus({ kind: "pending", text: "Davet gönderiliyor…" });
    try {
      try {
        await sendInvite(trimmed);
      } catch (err) {
        if (graphqlErrorCode(err) !== "CONSENT_REQUIRED") {
          throw err;
        }
        setStatus({ kind: "pending", text: "Onaylar tamamlanıyor, davet yeniden gönderiliyor…" });
        await ensureInviteConsents();
        await sendInvite(trimmed);
      }
      setEmail("");
      setStatus({
        kind: "success",
        text: `Davet ${trimmed} adresine gönderildi. Gelen kutusu ve spam klasörünü kontrol edin.`,
      });
      await load({ silent: true });
    } catch (err) {
      setStatus({ kind: "error", text: authErrorMessage(err, "Davet gönderilemedi") });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AppShell
      title="Üyeler"
      kicker="Organizasyon"
      orgId={orgId}
      restricted={view === "unauthorized"}
      description="Davet gönderin; roller OWNER, ADMIN, ANALYST veya VIEWER olabilir."
    >
      {view === "loading" && members.length === 0 && <AsyncView state="loading" />}
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
      {org && isPersonal && (
        <div className="alert-warn mt-6 max-w-lg space-y-3">
          <p>Kişisel çalışma alanına başka üye eklenemez. Ekip davet etmek için yeni bir organizasyon oluşturun.</p>
          <Link href="/org/create" className="btn-primary">
            Organizasyon oluştur
          </Link>
        </div>
      )}
      {org && !isPersonal && view !== "unauthorized" && (
          <form noValidate onSubmit={invite} className="card mt-6 max-w-lg space-y-3">
            <h2 className="font-display text-xl">Davet gönder</h2>
            <div
              role="status"
              aria-live="polite"
              className={
                status.kind === "error"
                  ? "alert-error space-y-2"
                  : status.kind === "success"
                    ? "alert-success"
                    : status.kind === "pending"
                      ? "alert-info"
                      : "min-h-5 text-sm text-muted"
              }
            >
              <p>{status.text || "Davet sonucunu burada göreceksiniz."}</p>
              {status.kind === "error" && status.text.includes("uyumluluk") && (
                <Link href={`/org/${orgId}/compliance`} className="font-semibold underline">
                  Uyumluluk sayfasına git
                </Link>
              )}
            </div>
            <input
              className="input"
              type="text"
              inputMode="email"
              autoComplete="email"
              placeholder="E-posta"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={submitting}
            />
            <select className="input" value={role} onChange={(e) => setRole(e.target.value)} disabled={submitting}>
              <option value="ADMIN">ADMIN</option>
              <option value="ANALYST">ANALYST</option>
              <option value="VIEWER">VIEWER</option>
            </select>
            <button type="submit" className="btn-primary" disabled={submitting} aria-busy={submitting}>
              {submitting ? "Gönderiliyor…" : "Davet et"}
            </button>
          </form>
      )}
    </AppShell>
  );
}
